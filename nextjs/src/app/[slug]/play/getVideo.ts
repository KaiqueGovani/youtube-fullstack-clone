import { VideoModel } from "../../../models";

export async function getVideo(slug: string): Promise<VideoModel> {
  const response = await fetch(`${process.env.DJANGO_API_URL}/videos/${slug}`, {
    next: {
      tags: [`video-${slug}`],
    },
  });
  return response.json();
}

