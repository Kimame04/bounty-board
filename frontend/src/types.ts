export interface User {
  id: number;
  username: string;
  xp: number;
  level: number;
  title: string;
  created_at: string;
}

export interface Topic {
  id: number;
  name: string;
  description: string;
  post_count?: number;
  created_at: string;
}

export interface Post {
  id: number;
  topic_id: number;
  author_id: number;
  author_name: string;
  author_level: number;
  title: string;
  content: string;
  is_resolved: boolean;
  accepted_comment_id?: number | null;
  comment_count: number;
  created_at: string;
  updated_at: string;
}

export interface Comment {
  id: number;
  post_id: number;
  author_id: number;
  author_name: string;
  author_level: number;
  content: string;
  upvotes: number;
  is_accepted: boolean;
  created_at: string;
  updated_at: string;
}

export interface PartyRewardSummary {
  user_id: number;
  username: string;
  xp_awarded: number;
  reason: string;
  new_xp: number;
  new_level: number;
  new_title: string;
}

export interface ResolveResponse {
  post: Post;
  rewards: PartyRewardSummary[];
  message: string;
}
