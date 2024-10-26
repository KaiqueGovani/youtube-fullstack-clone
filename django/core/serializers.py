from rest_framework import serializers
from core.models import Video
from django.conf import settings

class VideoSerializer(serializers.ModelSerializer):
    id = serializers.IntegerField()
    title = serializers.CharField()
    description = serializers.CharField()
    slug = serializers.CharField()
    published_at = serializers.DateTimeField()
    views = serializers.IntegerField(source='num_views')
    likes = serializers.IntegerField(source='num_likes')
    tags = serializers.PrimaryKeyRelatedField(many=True, read_only=True)
    thumbnail = serializers.SerializerMethodField()
    video_url = serializers.SerializerMethodField()


    def get_thumbnail(self, obj):
        assets_url = settings.ASSETS_URL
        return f'http://host.docker.internal:9000/media/uploads{obj.thumbnail if f"{obj.thumbnail}".startswith("/") else f"/{obj.thumbnail}"}'
    
    def get_video_url(self, obj):
        assets_url = settings.ASSETS_URL
        if (obj.video_media.video_path.startswith("/media/uploads")):
            path = f'http://host.docker.internal:9000{obj.video_media.video_path}/mpeg-dash/output.mpd'
        else:
            path = f'http://host.docker.internal:9000/media/uploads{obj.video_media.video_path}'
        if hasattr(obj, 'video_media'):
            return path
        return None
    
    class Meta:
        model = Video
        fields = ['id','title', 'description', 'slug', 'published_at', 'views', 'likes', 'tags', 'thumbnail', 'video_url']
    