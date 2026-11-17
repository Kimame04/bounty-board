import React, { useState, useEffect } from 'react';
import {
  AppBar,
  Toolbar,
  Typography,
  Container,
  Box,
  Button,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Card,
  CardActionArea,
  Chip,
  IconButton,
  Divider,
  Alert,
  Tabs,
  Tab,
  Stack,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';

import { api } from './api/client';
import { User, Topic, Post, Comment, PartyRewardSummary } from './types';
import { AvatarSprite } from './components/AvatarSprite';

export default function App() {
  // Current Adventurer Auth State
  const [user, setUser] = useState<User | null>(() => {
    const saved = localStorage.getItem('bounty_user');
    return saved ? JSON.parse(saved) : null;
  });

  const [loginInput, setLoginInput] = useState('');
  const [loginDialogOpen, setLoginDialogOpen] = useState(false);

  // App Navigation State
  const [topics, setTopics] = useState<Topic[]>([]);
  const [selectedTopic, setSelectedTopic] = useState<Topic | null>(null);
  const [posts, setPosts] = useState<Post[]>([]);
  const [selectedPost, setSelectedPost] = useState<Post | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [statusFilter, setStatusFilter] = useState<'all' | 'open' | 'resolved'>('all');

  // Modals & Forms
  const [newTopicName, setNewTopicName] = useState('');
  const [newTopicDesc, setNewTopicDesc] = useState('');
  const [topicDialogOpen, setTopicDialogOpen] = useState(false);

  const [newPostTitle, setNewPostTitle] = useState('');
  const [newPostContent, setNewPostContent] = useState('');
  const [postDialogOpen, setPostDialogOpen] = useState(false);

  const [commentInput, setCommentInput] = useState('');
  const [leaderboardOpen, setLeaderboardOpen] = useState(false);
  const [leaderboard, setLeaderboard] = useState<User[]>([]);

  // Party XP Celebration Dialog
  const [rewardCelebration, setRewardCelebration] = useState<{
    open: boolean;
    rewards: PartyRewardSummary[];
  }>({ open: false, rewards: [] });

  // Initial load
  useEffect(() => {
    loadTopics();
  }, []);

  const loadTopics = async () => {
    try {
      const data = await api.listTopics();
      setTopics(data);
    } catch (err) {
      console.error(err);
    }
  };

  const loadPosts = async (topicId: number) => {
    try {
      const filter = statusFilter === 'all' ? undefined : statusFilter;
      const data = await api.listPostsByTopic(topicId, filter);
      setPosts(data);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    if (selectedTopic) {
      loadPosts(selectedTopic.id);
    }
  }, [selectedTopic, statusFilter]);

  const loadComments = async (postId: number) => {
    try {
      const data = await api.listComments(postId);
      setComments(data);
    } catch (err) {
      console.error(err);
    }
  };

  const handleLogin = async () => {
    if (!loginInput.trim()) return;
    try {
      const loggedIn = await api.login(loginInput.trim());
      setUser(loggedIn);
      localStorage.setItem('bounty_user', JSON.stringify(loggedIn));
      setLoginDialogOpen(false);
      setLoginInput('');
    } catch (err: any) {
      alert(err.message || 'Login failed');
    }
  };

  const handleLogout = () => {
    setUser(null);
    localStorage.removeItem('bounty_user');
  };

  const handleCreateTopic = async () => {
    if (!newTopicName.trim()) return;
    try {
      await api.createTopic(newTopicName.trim(), newTopicDesc.trim());
      setTopicDialogOpen(false);
      setNewTopicName('');
      setNewTopicDesc('');
      loadTopics();
    } catch (err: any) {
      alert(err.message || 'Failed to create Guild Hall');
    }
  };

  const handleCreatePost = async () => {
    if (!user) {
      setLoginDialogOpen(true);
      return;
    }
    if (!selectedTopic || !newPostTitle.trim() || !newPostContent.trim()) return;
    try {
      await api.createPost(selectedTopic.id, user.id, newPostTitle.trim(), newPostContent.trim());
      setPostDialogOpen(false);
      setNewPostTitle('');
      setNewPostContent('');
      loadPosts(selectedTopic.id);
      loadTopics();
    } catch (err: any) {
      alert(err.message || 'Failed to post quest');
    }
  };

  const handleCreateComment = async () => {
    if (!user) {
      setLoginDialogOpen(true);
      return;
    }
    if (!selectedPost || !commentInput.trim()) return;
    try {
      await api.createComment(selectedPost.id, user.id, commentInput.trim());
      setCommentInput('');
      loadComments(selectedPost.id);
      // reload post comment count
      const updatedPost = await api.getPost(selectedPost.id);
      setSelectedPost(updatedPost);
    } catch (err: any) {
      alert(err.message || 'Failed to post comment');
    }
  };

  const handleUpvoteComment = async (commentId: number) => {
    try {
      await api.upvoteComment(commentId);
      if (selectedPost) loadComments(selectedPost.id);
    } catch (err: any) {
      alert(err.message || 'Failed to upvote');
    }
  };

  const handleResolveQuest = async (acceptedCommentId?: number) => {
    if (!user || !selectedPost) return;
    try {
      const res = await api.resolveQuest(selectedPost.id, user.id, acceptedCommentId);
      setSelectedPost(res.post);
      loadComments(selectedPost.id);
      setRewardCelebration({ open: true, rewards: res.rewards });

      // Refresh current user stats if they participated
      const refreshedUser = await api.getUser(user.username);
      setUser(refreshedUser);
      localStorage.setItem('bounty_user', JSON.stringify(refreshedUser));
    } catch (err: any) {
      alert(err.message || 'Failed to resolve quest');
    }
  };

  const openLeaderboard = async () => {
    try {
      const data = await api.getLeaderboard();
      setLeaderboard(data);
      setLeaderboardOpen(true);
    } catch (err) {
      console.error(err);
    }
  };

  // Find unique party members for the current quest
  const partyMembers = React.useMemo(() => {
    const map = new Map<number, { name: string; level: number }>();
    comments.forEach((c) => {
      if (!map.has(c.author_id)) {
        map.set(c.author_id, { name: c.author_name, level: c.author_level });
      }
    });
    return Array.from(map.entries()).map(([id, data]) => ({ id, ...data }));
  }, [comments]);

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#0f172a', color: '#f8fafc', pb: 8 }}>
      {/* Navbar */}
      <AppBar position="sticky" sx={{ bgcolor: '#1e293b', borderBottom: '1px solid #334155' }} elevation={0}>
        <Toolbar sx={{ justifyContent: 'space-between' }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Typography variant="h6" sx={{ fontWeight: 700, display: 'flex', alignItems: 'center', gap: 1 }}>
              <span className="pixel-font" style={{ fontSize: '1.1rem', color: '#f59e0b' }}>⚔️ BOUNTY BOARD</span>
            </Typography>
          </Stack>

          <Stack direction="row" spacing={2} alignItems="center">
            <Button
              startIcon={<EmojiEventsIcon sx={{ color: '#f59e0b' }} />}
              variant="outlined"
              size="small"
              onClick={openLeaderboard}
              sx={{ borderColor: '#475569', color: '#cbd5e1' }}
            >
              Leaderboard
            </Button>

            {user ? (
              <Stack direction="row" spacing={1.5} alignItems="center">
                <AvatarSprite username={user.username} level={user.level} size={38} />
                <Box>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
                    {user.username}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#f59e0b', fontWeight: 700 }}>
                    Lvl {user.level} • {user.xp} XP
                  </Typography>
                </Box>
                <Button size="small" color="inherit" onClick={handleLogout} sx={{ opacity: 0.7 }}>
                  Logout
                </Button>
              </Stack>
            ) : (
              <Button variant="contained" color="primary" size="small" onClick={() => setLoginDialogOpen(true)}>
                Enter Guild (Login)
              </Button>
            )}
          </Stack>
        </Toolbar>
      </AppBar>

      {/* Main Content View Container */}
      <Container maxWidth="lg" sx={{ mt: 4 }}>
        {/* VIEW 1: GUILD HALLS / TOPICS */}
        {!selectedTopic && (
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
              <Box>
                <Typography variant="h4" sx={{ fontWeight: 700 }}>
                  Guild Halls 🏰
                </Typography>
                <Typography variant="body2" sx={{ color: '#94a3b8' }}>
                  Choose a realm to browse open bounties and trade technical clues.
                </Typography>
              </Box>
              <Button
                variant="contained"
                color="primary"
                startIcon={<AddCircleOutlineIcon />}
                onClick={() => (user ? setTopicDialogOpen(true) : setLoginDialogOpen(true))}
              >
                New Guild Hall
              </Button>
            </Box>

            {topics.length === 0 ? (
              <Alert severity="info" sx={{ bgcolor: '#1e293b', color: '#cbd5e1' }}>
                No Guild Halls established yet. Be the first adventurer to found one!
              </Alert>
            ) : (
              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2.5 }}>
                {topics.map((t) => (
                  <Card
                    key={t.id}
                    sx={{
                      bgcolor: '#1e293b',
                      border: '1px solid #334155',
                      borderRadius: 2,
                      transition: 'transform 0.15s, border-color 0.15s',
                      '&:hover': { borderColor: '#f59e0b', transform: 'translateY(-2px)' },
                    }}
                  >
                    <CardActionArea onClick={() => setSelectedTopic(t)} sx={{ p: 2.5 }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                        <Typography variant="h6" sx={{ fontWeight: 600, color: '#f8fafc' }}>
                          #{t.name}
                        </Typography>
                        <Chip
                          label={`${t.post_count || 0} Quests`}
                          size="small"
                          sx={{ bgcolor: '#334155', color: '#cbd5e1', fontWeight: 600 }}
                        />
                      </Box>
                      <Typography variant="body2" sx={{ color: '#94a3b8', mt: 1 }}>
                        {t.description || 'A gathering place for curious minds.'}
                      </Typography>
                    </CardActionArea>
                  </Card>
                ))}
              </Box>
            )}
          </Box>
        )}

        {/* VIEW 2: QUEST BOARD (POSTS UNDER TOPIC) */}
        {selectedTopic && !selectedPost && (
          <Box>
            <Button
              startIcon={<ArrowBackIcon />}
              onClick={() => setSelectedTopic(null)}
              sx={{ color: '#94a3b8', mb: 2 }}
            >
              Back to Guild Halls
            </Button>

            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
              <Box>
                <Typography variant="h4" sx={{ fontWeight: 700 }}>
                  #{selectedTopic.name}
                </Typography>
                <Typography variant="body2" sx={{ color: '#94a3b8' }}>
                  {selectedTopic.description}
                </Typography>
              </Box>
              <Button
                variant="contained"
                color="primary"
                startIcon={<AddCircleOutlineIcon />}
                onClick={() => (user ? setPostDialogOpen(true) : setLoginDialogOpen(true))}
              >
                Post a Bounty
              </Button>
            </Box>

            {/* Filter Tabs */}
            <Tabs
              value={statusFilter}
              onChange={(_, val) => setStatusFilter(val)}
              sx={{ borderBottom: '1px solid #334155', mb: 3 }}
              textColor="primary"
              indicatorColor="primary"
            >
              <Tab label="All Bounties" value="all" sx={{ color: '#94a3b8', fontWeight: 600 }} />
              <Tab label="Open Quests (⚔️)" value="open" sx={{ color: '#94a3b8', fontWeight: 600 }} />
              <Tab label="Solved (🛡️)" value="resolved" sx={{ color: '#94a3b8', fontWeight: 600 }} />
            </Tabs>

            {posts.length === 0 ? (
              <Alert severity="info" sx={{ bgcolor: '#1e293b', color: '#cbd5e1' }}>
                No bounties match this filter. Post the first quest to recruit party members!
              </Alert>
            ) : (
              <Stack spacing={2}>
                {posts.map((p) => (
                  <Card
                    key={p.id}
                    sx={{
                      bgcolor: '#1e293b',
                      border: p.is_resolved ? '1px solid #065f46' : '1px solid #334155',
                      borderRadius: 2,
                      '&:hover': { borderColor: p.is_resolved ? '#10b981' : '#f59e0b' },
                    }}
                  >
                    <CardActionArea onClick={() => { setSelectedPost(p); loadComments(p.id); }} sx={{ p: 2.5 }}>
                      <Stack direction="row" spacing={2} alignItems="center">
                        <AvatarSprite username={p.author_name} level={p.author_level} size={48} />
                        <Box sx={{ flexGrow: 1 }}>
                          <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
                            {p.is_resolved ? (
                              <Chip
                                icon={<CheckCircleIcon sx={{ fontSize: '1rem !important', color: '#10b981' }} />}
                                label="Solved"
                                size="small"
                                sx={{ bgcolor: '#064e3b', color: '#6ee7b7', fontWeight: 600 }}
                              />
                            ) : (
                              <Chip
                                label="⚔️ Open Quest"
                                size="small"
                                sx={{ bgcolor: '#78350f', color: '#fde68a', fontWeight: 600 }}
                              />
                            )}
                            <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                              By {p.author_name} (Lvl {p.author_level})
                            </Typography>
                          </Stack>
                          <Typography variant="h6" sx={{ fontWeight: 600, color: '#f8fafc' }}>
                            {p.title}
                          </Typography>
                          <Typography variant="body2" sx={{ color: '#94a3b8', mt: 0.5 }}>
                            {p.content.length > 140 ? p.content.slice(0, 140) + '...' : p.content}
                          </Typography>
                        </Box>
                        <Chip
                          label={`${p.comment_count} Dispatches`}
                          size="small"
                          sx={{ bgcolor: '#334155', color: '#cbd5e1' }}
                        />
                      </Stack>
                    </CardActionArea>
                  </Card>
                ))}
              </Stack>
            )}
          </Box>
        )}

        {/* VIEW 3: QUEST DETAIL & PARTY DISPATCHES */}
        {selectedTopic && selectedPost && (
          <Box>
            <Button
              startIcon={<ArrowBackIcon />}
              onClick={() => { setSelectedPost(null); loadPosts(selectedTopic.id); }}
              sx={{ color: '#94a3b8', mb: 2 }}
            >
              Back to #{selectedTopic.name}
            </Button>

            {/* Post Card */}
            <Card sx={{ bgcolor: '#1e293b', border: '1px solid #334155', borderRadius: 2, p: 3, mb: 3 }}>
              <Stack direction="row" spacing={2} alignItems="flex-start">
                <AvatarSprite username={selectedPost.author_name} level={selectedPost.author_level} size={56} />
                <Box sx={{ flexGrow: 1 }}>
                  <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                    {selectedPost.is_resolved ? (
                      <Chip
                        icon={<CheckCircleIcon sx={{ fontSize: '1rem !important', color: '#10b981' }} />}
                        label="Quest Solved"
                        size="small"
                        sx={{ bgcolor: '#064e3b', color: '#6ee7b7', fontWeight: 600 }}
                      />
                    ) : (
                      <Chip
                        label="⚔️ Open Quest"
                        size="small"
                        sx={{ bgcolor: '#78350f', color: '#fde68a', fontWeight: 600 }}
                      />
                    )}
                    <Typography variant="body2" sx={{ color: '#94a3b8' }}>
                      Posted by <strong>{selectedPost.author_name}</strong> (Lvl {selectedPost.author_level})
                    </Typography>
                  </Stack>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: '#f8fafc', mb: 1.5 }}>
                    {selectedPost.title}
                  </Typography>
                  <Typography variant="body1" sx={{ color: '#cbd5e1', whiteSpace: 'pre-wrap' }}>
                    {selectedPost.content}
                  </Typography>
                </Box>
              </Stack>

              {/* Party Members Row */}
              <Divider sx={{ my: 2.5, borderColor: '#334155' }} />
              <Box>
                <Typography variant="caption" sx={{ color: '#94a3b8', fontWeight: 700, textTransform: 'uppercase' }}>
                  Adventuring Party ({partyMembers.length} active contributors)
                </Typography>
                {partyMembers.length === 0 ? (
                  <Typography variant="body2" sx={{ color: '#64748b', mt: 0.5 }}>
                    No party members have joined yet. Be the first to dispatch a clue!
                  </Typography>
                ) : (
                  <Stack direction="row" spacing={1.5} sx={{ mt: 1, flexWrap: 'wrap' }}>
                    {partyMembers.map((m) => (
                      <Chip
                        key={m.id}
                        avatar={<AvatarSprite username={m.name} level={m.level} size={28} />}
                        label={`${m.name} (Lvl ${m.level})`}
                        sx={{ bgcolor: '#0f172a', color: '#e2e8f0', border: '1px solid #334155' }}
                      />
                    ))}
                  </Stack>
                )}
              </Box>
            </Card>

            {/* Dispatches / Solutions Section */}
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
              Party Dispatches & Solutions ({comments.length})
            </Typography>

            <Stack spacing={2} sx={{ mb: 4 }}>
              {comments.map((c) => (
                <Card
                  key={c.id}
                  sx={{
                    bgcolor: c.is_accepted ? '#064e3b22' : '#1e293b',
                    border: c.is_accepted ? '2px solid #10b981' : '1px solid #334155',
                    borderRadius: 2,
                    p: 2.5,
                  }}
                >
                  <Stack direction="row" spacing={2} alignItems="flex-start">
                    <AvatarSprite username={c.author_name} level={c.author_level} size={42} />
                    <Box sx={{ flexGrow: 1 }}>
                      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                          {c.author_name}
                        </Typography>
                        <Typography variant="caption" sx={{ color: '#f59e0b' }}>
                          Lvl {c.author_level}
                        </Typography>
                        {c.is_accepted && (
                          <Chip
                            icon={<CheckCircleIcon sx={{ fontSize: '0.9rem !important', color: '#10b981' }} />}
                            label="Master Solution"
                            size="small"
                            sx={{ bgcolor: '#064e3b', color: '#6ee7b7', fontWeight: 600, height: 22 }}
                          />
                        )}
                      </Stack>
                      <Typography variant="body2" sx={{ color: '#e2e8f0', whiteSpace: 'pre-wrap' }}>
                        {c.content}
                      </Typography>
                    </Box>

                    {/* Upvote & Action Buttons */}
                    <Stack alignItems="center" spacing={1}>
                      <IconButton
                        size="small"
                        onClick={() => handleUpvoteComment(c.id)}
                        sx={{ color: c.upvotes > 0 ? '#f59e0b' : '#64748b' }}
                      >
                        <ThumbUpIcon fontSize="small" />
                      </IconButton>
                      <Typography variant="caption" sx={{ fontWeight: 700, color: '#cbd5e1' }}>
                        {c.upvotes}
                      </Typography>

                      {/* Author Crown Solution Button */}
                      {user && user.id === selectedPost.author_id && !selectedPost.is_resolved && (
                        <Button
                          size="small"
                          variant="contained"
                          color="secondary"
                          onClick={() => handleResolveQuest(c.id)}
                          sx={{ textTransform: 'none', fontSize: '0.75rem', mt: 1 }}
                        >
                          Crown Solution & Reward Party ✨
                        </Button>
                      )}
                    </Stack>
                  </Stack>
                </Card>
              ))}
            </Stack>

            {/* Add Solution Input */}
            <Card sx={{ bgcolor: '#1e293b', border: '1px solid #334155', borderRadius: 2, p: 2.5 }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 1 }}>
                Send a Dispatch / Solution 📜
              </Typography>
              <TextField
                fullWidth
                multiline
                rows={3}
                placeholder={user ? "Write your solution to help the party..." : "Login to write a solution"}
                disabled={!user}
                value={commentInput}
                onChange={(e) => setCommentInput(e.target.value)}
                sx={{ bgcolor: '#0f172a', borderRadius: 1, mb: 2 }}
              />
              <Button
                variant="contained"
                color="primary"
                disabled={!user || !commentInput.trim()}
                onClick={handleCreateComment}
              >
                Submit Dispatch
              </Button>
            </Card>
          </Box>
        )}
      </Container>

      {/* LOGIN DIALOG (When2meet style) */}
      <Dialog open={loginDialogOpen} onClose={() => setLoginDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ bgcolor: '#1e293b', color: '#f8fafc', fontWeight: 700 }}>
          Enter the Guild ⚔️
        </DialogTitle>
        <DialogContent sx={{ bgcolor: '#1e293b', pt: 1 }}>
          <Typography variant="body2" sx={{ color: '#94a3b8', mb: 2 }}>
            Authenticate solely by your adventurer handle (like When2meet). New handles are registered automatically!
          </Typography>
          <TextField
            autoFocus
            fullWidth
            label="Adventurer Name"
            value={loginInput}
            onChange={(e) => setLoginInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
          />
        </DialogContent>
        <DialogActions sx={{ bgcolor: '#1e293b', p: 2 }}>
          <Button onClick={() => setLoginDialogOpen(false)} sx={{ color: '#94a3b8' }}>
            Cancel
          </Button>
          <Button variant="contained" color="primary" onClick={handleLogin}>
            Enter
          </Button>
        </DialogActions>
      </Dialog>

      {/* CREATE TOPIC DIALOG */}
      <Dialog open={topicDialogOpen} onClose={() => setTopicDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ bgcolor: '#1e293b', color: '#f8fafc', fontWeight: 700 }}>
          Found a New Guild Hall 🏰
        </DialogTitle>
        <DialogContent sx={{ bgcolor: '#1e293b', pt: 1 }}>
          <TextField
            autoFocus
            fullWidth
            label="Guild Hall Name (e.g. Go-Spells)"
            value={newTopicName}
            onChange={(e) => setNewTopicName(e.target.value)}
            sx={{ mb: 2, mt: 1 }}
          />
          <TextField
            fullWidth
            multiline
            rows={2}
            label="Realm Description"
            value={newTopicDesc}
            onChange={(e) => setNewTopicDesc(e.target.value)}
          />
        </DialogContent>
        <DialogActions sx={{ bgcolor: '#1e293b', p: 2 }}>
          <Button onClick={() => setTopicDialogOpen(false)} sx={{ color: '#94a3b8' }}>
            Cancel
          </Button>
          <Button variant="contained" color="primary" onClick={handleCreateTopic}>
            Found Realm
          </Button>
        </DialogActions>
      </Dialog>

      {/* CREATE POST DIALOG */}
      <Dialog open={postDialogOpen} onClose={() => setPostDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ bgcolor: '#1e293b', color: '#f8fafc', fontWeight: 700 }}>
          Post a Bounty Quest ⚔️
        </DialogTitle>
        <DialogContent sx={{ bgcolor: '#1e293b', pt: 1 }}>
          <TextField
            autoFocus
            fullWidth
            label="Quest Title / Issue"
            value={newPostTitle}
            onChange={(e) => setNewPostTitle(e.target.value)}
            sx={{ mb: 2, mt: 1 }}
          />
          <TextField
            fullWidth
            multiline
            rows={4}
            label="Quest Details & Context"
            value={newPostContent}
            onChange={(e) => setNewPostContent(e.target.value)}
          />
        </DialogContent>
        <DialogActions sx={{ bgcolor: '#1e293b', p: 2 }}>
          <Button onClick={() => setPostDialogOpen(false)} sx={{ color: '#94a3b8' }}>
            Cancel
          </Button>
          <Button variant="contained" color="primary" onClick={handleCreatePost}>
            Pin to Board
          </Button>
        </DialogActions>
      </Dialog>

      {/* LEADERBOARD DIALOG */}
      <Dialog open={leaderboardOpen} onClose={() => setLeaderboardOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ bgcolor: '#1e293b', color: '#f8fafc', fontWeight: 700, display: 'flex', alignItems: 'center', gap: 1 }}>
          <EmojiEventsIcon sx={{ color: '#f59e0b' }} /> Guild Hall of Fame
        </DialogTitle>
        <DialogContent sx={{ bgcolor: '#1e293b', p: 0 }}>
          <Stack divider={<Divider sx={{ borderColor: '#334155' }} />}>
            {leaderboard.map((u, idx) => (
              <Box key={u.id} sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 2 }}>
                <Typography sx={{ fontWeight: 700, width: 24, color: idx < 3 ? '#f59e0b' : '#64748b' }}>
                  #{idx + 1}
                </Typography>
                <AvatarSprite username={u.username} level={u.level} size={40} />
                <Box sx={{ flexGrow: 1 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    {u.username}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                    {u.title}
                  </Typography>
                </Box>
                <Box sx={{ textAlign: 'right' }}>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#f59e0b' }}>
                    {u.xp} XP
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#64748b' }}>
                    Level {u.level}
                  </Typography>
                </Box>
              </Box>
            ))}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ bgcolor: '#1e293b', p: 2 }}>
          <Button onClick={() => setLeaderboardOpen(false)} variant="contained" color="primary">
            Close
          </Button>
        </DialogActions>
      </Dialog>

      {/* PARTY XP REWARD CELEBRATION DIALOG */}
      <Dialog open={rewardCelebration.open} onClose={() => setRewardCelebration({ open: false, rewards: [] })} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ bgcolor: '#1e293b', color: '#10b981', fontWeight: 700, textAlign: 'center', pt: 3 }}>
          🎉 Quest Solved & Cleared!
        </DialogTitle>
        <DialogContent sx={{ bgcolor: '#1e293b', textAlign: 'center' }}>
          <Typography variant="body2" sx={{ color: '#cbd5e1', mb: 2 }}>
            The quest has been crowned! Party XP has been distributed:
          </Typography>
          <Stack spacing={1.5}>
            {rewardCelebration.rewards.map((r, i) => (
              <Box
                key={i}
                sx={{
                  bgcolor: '#0f172a',
                  p: 1.5,
                  borderRadius: 2,
                  border: '1px solid #334155',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1.5,
                }}
              >
                <AvatarSprite username={r.username} level={r.new_level} size={36} />
                <Box sx={{ textAlign: 'left', flexGrow: 1 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    {r.username}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                    {r.reason === 'accepted_solution' ? 'Master Solver (+50 XP)' : r.reason === 'party_participant' ? 'Party Contributor (+15 XP)' : 'Quest Author (+10 XP)'}
                  </Typography>
                </Box>
                <Typography variant="body2" sx={{ fontWeight: 700, color: '#10b981' }}>
                  +{r.xp_awarded} XP
                </Typography>
              </Box>
            ))}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ bgcolor: '#1e293b', p: 2, justifyContent: 'center' }}>
          <Button
            variant="contained"
            color="secondary"
            onClick={() => setRewardCelebration({ open: false, rewards: [] })}
          >
            Huzzah!
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
