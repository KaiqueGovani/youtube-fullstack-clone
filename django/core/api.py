from core.models import Video
from rest_framework.response import Response
from rest_framework.decorators import api_view

from core.serializers import VideoSerializer
from django.db.models import Q

@api_view(['GET'])
def videos_list(request):
    q = request.query_params.get('q')
    if q:
        videos = Video.objects.filter(Q(title__icontains=q) | Q(description__icontains=q) | Q(tags__name__icontains=q))
    else:
        videos = Video.objects.all()
    query = videos.filter(is_published=True).order_by('-published_at').distinct()
    serializer = VideoSerializer(query, many=True)
    return Response(serializer.data)
