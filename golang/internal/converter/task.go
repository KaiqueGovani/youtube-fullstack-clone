package converter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"fullstack-youtube-clone/internal/rabbitmq"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/streadway/amqp"
)

type VideoConverter struct {
	db             *sql.DB
	rabbitmqClient *rabbitmq.RabbitClient
}

func NewVideoConverter(rabbitmqClient *rabbitmq.RabbitClient, db *sql.DB) *VideoConverter {
	return &VideoConverter{
		rabbitmqClient: rabbitmqClient,
		db:             db,
	}
}

type VideoTask struct {
	VideoID int    `json:"video_id"`
	Path    string `json:"path"`
}

func (vc *VideoConverter) Handle(d amqp.Delivery, conversionExch, confirmationKey, queueName string) {
	var task VideoTask
	if err := json.Unmarshal(d.Body, &task); err != nil {
		vc.logError(task, "failed to unmarshal task", err)
		return
	}

	if IsProcessed(vc.db, task.VideoID) {
		slog.Warn("Video already processed", slog.Int("video_id", task.VideoID))
		d.Ack(false)
		return
	}

	err := vc.processVideo(&task)
	if err != nil {
		vc.logError(task, "failed to process video", err)
		return
	}

	err = MarkProcessed(vc.db, task.VideoID)
	if err != nil {
		vc.logError(task, "failed to mark video as processed", err)
		return
	}
	d.Ack(false)
	slog.Info("Video processed", slog.Int("video_id", task.VideoID))

	confirmationMessage := []byte(fmt.Sprintf(`{"video_id": %d, "path": "%s"}`, task.VideoID, task.Path))
	err = vc.rabbitmqClient.PublishMessage(conversionExch, confirmationKey, queueName, confirmationMessage)
	if err != nil {
		vc.logError(task, "failed to publish confirmation message", err)
		return
	}
}

func (vc *VideoConverter) processVideo(task *VideoTask) error {
	mergedFile := filepath.Join(task.Path, "merged.mp4")
	mpegDashPath := filepath.Join(task.Path, "mpeg-dash")

	slog.Info("Merging chunks", slog.String("path", task.Path))
	err := vc.mergeChunks(task.Path, mergedFile)
	if err != nil {
		vc.logError(*task, "failed to merge chunks", err)
		return err
	}

	err = os.MkdirAll(mpegDashPath, os.ModePerm)
	if err != nil {
		vc.logError(*task, "failed to create mpeg-dash directory", err)
		return err
	}

	slog.Info("Converting video to mpeg-dash", slog.String("path", task.Path))
	ffmpegCmd := exec.Command(
		"ffmpeg", "-i", mergedFile,
		"-f", "dash",
		filepath.Join(mpegDashPath, "output.mpd"),
	)

	output, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		vc.logError(*task, "failed to convert video to mpeg-dash, output: "+string(output), err)
		return err
	}
	slog.Info("Video converted", slog.String("path", mpegDashPath))

	slog.Info("Removing merged file", slog.String("path", mergedFile))
	err = os.Remove(mergedFile)
	if err != nil {
		vc.logError(*task, "failed to remove merged file", err)
		return err
	}

	return nil
}

func (vc *VideoConverter) logError(task VideoTask, message string, err error) {
	errorData := map[string]any{
		"video_id": task.VideoID,
		"error":    message,
		"details":  err.Error(),
		"time":     time.Now(),
	}
	serializedError, _ := json.Marshal(errorData)
	slog.Error("Processing error", slog.String("error_details", string(serializedError)))

	RegisterError(vc.db, errorData, err)
}

func (vc *VideoConverter) extractNumber(filename string) int {
	re := regexp.MustCompile(`\d+`)
	numStr := re.FindString(filepath.Base(filename))

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}
	return num
}

func (vc *VideoConverter) mergeChunks(inputDir, outputFile string) error {
	chunks, err := filepath.Glob(filepath.Join(inputDir, "*.chunk"))
	if err != nil {
		return fmt.Errorf("failed to find chunks: %w", err)
	}

	sort.Slice(chunks, func(i, j int) bool {
		return vc.extractNumber(chunks[i]) < vc.extractNumber(chunks[j])
	})

	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer output.Close()

	for _, chunk := range chunks {
		input, err := os.Open(chunk)
		if err != nil {
			return fmt.Errorf("failed to open chunk: %v", err)
		}

		_, err = output.ReadFrom(input)
		if err != nil {
			return fmt.Errorf("failed to write chunk %s to merged file: %v", chunk, err)
		}

		input.Close()
	}

	return nil
}
