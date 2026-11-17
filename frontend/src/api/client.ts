import { Topic, Post, Comment, User, ResolveResponse } from '../types';

const API_BASE = '/api';

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${url}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options?.headers || {}),
    },
    ...options,
  });

  if (!res.ok) {
    let errorMsg = `HTTP Error ${res.status}`;
    try {
      const body = await res.json();
      if (body.error) errorMsg = body.error;
    } catch {
      // ignore json parse error
    }
    throw new Error(errorMsg);
  }

  return res.json();
}

export const api = {
  // Auth & Adventurers
  login: (username: string) =>
    fetchJSON<User>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username }),
    }),

  getUser: (username: string) => fetchJSON<User>(`/users?username=${encodeURIComponent(username)}`),

  getLeaderboard: () => fetchJSON<User[]>('/leaderboard'),

  // Topics (Guild Halls)
  listTopics: () => fetchJSON<Topic[]>('/topics'),

  createTopic: (name: string, description: string) =>
    fetchJSON<Topic>('/topics', {
      method: 'POST',
      body: JSON.stringify({ name, description }),
    }),

  deleteTopic: (id: number) =>
    fetchJSON<{ message: string }>(`/topics/${id}`, { method: 'DELETE' }),

  // Posts (Quests)
  listPostsByTopic: (topicId: number, status?: 'open' | 'resolved') => {
    const q = status ? `?status=${status}` : '';
    return fetchJSON<Post[]>(`/topics/${topicId}/posts${q}`);
  },

  getPost: (id: number) => fetchJSON<Post>(`/posts/${id}`),

  createPost: (topicId: number, authorId: number, title: string, content: string) =>
    fetchJSON<Post>(`/topics/${topicId}/posts`, {
      method: 'POST',
      body: JSON.stringify({ author_id: authorId, title, content }),
    }),

  updatePost: (id: number, title: string, content: string) =>
    fetchJSON<Post>(`/posts/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ title, content }),
    }),

  deletePost: (id: number) =>
    fetchJSON<{ message: string }>(`/posts/${id}`, { method: 'DELETE' }),

  resolveQuest: (postId: number, requesterId: number, acceptedCommentId?: number) =>
    fetchJSON<ResolveResponse>(`/posts/${postId}/resolve`, {
      method: 'POST',
      body: JSON.stringify({
        requester_id: requesterId,
        accepted_comment_id: acceptedCommentId,
      }),
    }),

  // Comments (Solutions)
  listComments: (postId: number) => fetchJSON<Comment[]>(`/posts/${postId}/comments`),

  createComment: (postId: number, authorId: number, content: string) =>
    fetchJSON<Comment>(`/posts/${postId}/comments`, {
      method: 'POST',
      body: JSON.stringify({ author_id: authorId, content }),
    }),

  upvoteComment: (commentId: number) =>
    fetchJSON<{ id: number; upvotes: number }>(`/comments/${commentId}/upvote`, {
      method: 'POST',
    }),

  deleteComment: (commentId: number) =>
    fetchJSON<{ message: string }>(`/comments/${commentId}`, { method: 'DELETE' }),
};
