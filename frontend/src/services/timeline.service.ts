import { Paginated } from '@/types/api.types';
import { TimelineItem, TimelineQuery } from '@/types/timeline.types';
import { http } from './api';

export const timelineService = {
  list(query: TimelineQuery): Promise<Paginated<TimelineItem>> {
    return http.get<Paginated<TimelineItem>>('/timeline', { params: query });
  },
};
