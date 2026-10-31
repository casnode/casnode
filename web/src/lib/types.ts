// The shapes the backend (the Go code in the repository root) answers with.

export interface User {
  id: number;
  username: string;
  name: string;
  avatar: string;
  title: string;
  admin: boolean;
  moderator: boolean;
  trustLevel: number;
  // "local" for a password here, "oauth" for a provider on /admin/logins, none for Casdoor
  loginType?: string;
  email?: string;
  bioRaw?: string;
  bioCooked?: string;
  location?: string;
  website?: string;
  locale?: string;
  createdAt?: string;
  lastSeenAt?: string;
  lastPostedAt?: string;
  suspendedTill?: string;
  suspendReason?: string;
  silencedTill?: string;
  topicCount?: number;
  postCount?: number;
  likesGiven?: number;
  likesReceived?: number;
  daysVisited?: number;
  postsReadCount?: number;
  topicsEntered?: number;
  emailLevel?: number;
  emailMessagesLevel?: number;
  // a group offered among users, for @mentions and message recipients
  isGroup?: boolean;
}

export type UserMap = Record<string, User>;

export interface Category {
  id: number;
  name: string;
  slug: string;
  description: string;
  color: string;
  textColor: string;
  parentId: number;
  position: number;
  topicCount: number;
  postCount: number;
  readRestricted: boolean;
  createRestricted: boolean;
  topicTemplate: string;
  topicsWeek: number;
  topicsMonth: number;
  subcategoryIds: number[] | null;
  permissions: { groupId: number; permissionType: number }[] | null;
}

export interface Tag {
  id: number;
  name: string;
  description?: string;
  topicCount: number;
}

export interface Topic {
  id: number;
  title: string;
  slug: string;
  categoryId: number;
  userId: number;
  archetype: "regular" | "private_message";
  excerpt?: string;
  imageUrl?: string;
  pinnedAt: string;
  pinnedGlobally: boolean;
  closed: boolean;
  archived: boolean;
  visible: boolean;
  deleted: boolean;
  views: number;
  postsCount: number;
  replyCount: number;
  likeCount: number;
  highestPostNumber: number;
  participantCount: number;
  wordCount: number;
  lastPostUserId: number;
  lastPostedAt: string;
  bumpedAt: string;
  createdAt: string;
  posterIds: number[] | null;
  tags: string[] | null;
  lastReadPostNumber: number;
  unreadPosts: number;
  unseen: boolean;
  notificationLevel: number;
  mergedIntoId?: number;
}

export interface TopicList {
  topics: Topic[];
  users: UserMap;
  more: boolean;
  page: number;
}

export interface Post {
  id: number;
  topicId: number;
  postNumber: number;
  userId: number;
  raw?: string;
  cooked: string;
  postType: number;
  actionCode?: string;
  replyToPostNumber: number;
  replyToUserId: number;
  replyCount: number;
  likeCount: number;
  version: number;
  editReason?: string;
  lastEditorId?: number;
  deleted: boolean;
  hidden: boolean;
  createdAt: string;
  updatedAt: string;
  liked: boolean;
  bookmarked: boolean;
  flagged: boolean;
  canEdit: boolean;
  canDelete: boolean;
  canRecover: boolean;
  polls?: Poll[];
}

export interface TopicDetails {
  createdBy: number;
  lastPoster: number;
  participants: { userId: number; postCount: number }[] | null;
  allowedUserIds: number[] | null;
  allowedGroups: Group[] | null;
  canMovePosts: boolean;
  canEdit: boolean;
  canReply: boolean;
  canModerate: boolean;
  canDelete: boolean;
  canRecover: boolean;
  canClose: boolean;
}

export interface TopicView {
  topic: Topic;
  category: Category | null;
  posts: Post[];
  stream: number[];
  users: UserMap;
  details: TopicDetails;
  suggested: Topic[];
  suggestedUsers: UserMap;
  readingTime: number;
}

// a single sign-on provider on the login page
export interface AuthProviderInfo {
  name: string;
  displayName: string;
  icon: string;
}

// a single sign-on provider as /admin/logins edits it
export interface AuthProvider {
  name: string;
  displayName: string;
  type: "oidc" | "oauth2";
  icon: string;
  clientId: string;
  clientSecret: string;
  issuer: string;
  authUrl: string;
  tokenUrl: string;
  userInfoUrl: string;
  scopes: string;
  idField: string;
  usernameField: string;
  nameField: string;
  emailField: string;
  avatarField: string;
  allowSignup: boolean;
  enabled: boolean;
  sortOrder: number;
}

export interface SiteInfo {
  settings: Record<string, string>;
  casdoor: { serverUrl: string; clientId: string; organizationName: string; appName: string };
  authProviders: AuthProviderInfo[];
  passwordReset: boolean;
  devSignin: boolean;
  categories: Category[];
  topTags: Tag[];
  currentUser: User | null;
  counts: Record<string, number>;
  defaultLocale: string;
}

export interface Notification {
  id: number;
  userId: number;
  notificationType: number;
  topicId: number;
  postNumber: number;
  actingUserId: number;
  topicTitle: string;
  slug: string;
  excerpt?: string;
  read: boolean;
  highPriority: boolean;
  data?: string;
  createdAt: string;
}

export interface UserPost {
  post: Post;
  topic: Topic | null;
}

export interface UserSummary {
  topReplies: UserPost[];
  topTopics: Topic[];
  mostLikedBy: { userId: number; count: number }[];
  mostLiked: { userId: number; count: number }[];
  mostRepliedTo: { userId: number; count: number }[];
  topCategories: { categoryId: number; topicCount: number; postCount: number }[];
  bookmarkCount: number;
  users: UserMap;
}

export interface SearchResult {
  posts: { id: number; topicId: number; postNumber: number; userId: number; blurb: string; likeCount: number; createdAt: string }[];
  topics: Record<string, Topic>;
  users: UserMap;
  categories: Category[];
  tags: Tag[];
  matchUsers: User[];
  more: boolean;
}

export interface Reviewable {
  id: number;
  postId: number;
  topicId: number;
  status: number;
  score: number;
  scores: { userId: number; actionType: number; message?: string; createdAt: string }[];
  createdAt: string;
  post?: Post;
  topic?: Topic;
}

export interface Upload {
  id: number;
  url: string;
  originalFilename: string;
  width: number;
  height: number;
  extension: string;
}

export interface ApiResponse<T, T2 = unknown> {
  status: "ok" | "error";
  msg: string;
  data: T;
  data2: T2;
}

export const NotificationLevel = { Muted: 0, Regular: 1, Tracking: 2, Watching: 3 } as const;

export const NotificationType = {
  Mentioned: 1,
  Replied: 2,
  Quoted: 3,
  Edited: 4,
  Liked: 5,
  PrivateMessage: 6,
  InvitedToPrivateMessage: 7,
  Posted: 9,
  GrantedBadge: 12,
  MovedPost: 14,
  GroupMentioned: 15,
  AdminMessage: 20,
  MembershipAccepted: 22,
  MembershipRequest: 23,
  ChatMention: 29,
  ChatMessage: 30,
} as const;

export const PostAction = { Bookmark: 1, Like: 2, OffTopic: 3, Inappropriate: 4, NotifyModerators: 7, Spam: 8 } as const;

export interface Group {
  id: number;
  name: string;
  fullName: string;
  bioRaw?: string;
  bioCooked?: string;
  automatic: boolean;
  visibility: number;
  mentionableLevel: number;
  messageableLevel: number;
  publicJoin: boolean;
  publicExit: boolean;
  allowRequests: boolean;
  title: string;
  userCount: number;
  createdAt: string;
  isMember?: boolean;
  isOwner?: boolean;
  requested?: boolean;
  canManage?: boolean;
  canMention?: boolean;
  canMessage?: boolean;
  requestCount?: number;
}

export interface GroupMember {
  user: User;
  owner: boolean;
  addedAt: string;
}

export interface Badge {
  id: number;
  name: string;
  code: string;
  description: string;
  longDescription: string;
  icon: string;
  imageUrl: string;
  badgeType: number;
  grouping: string;
  multipleGrant: boolean;
  allowTitle: boolean;
  enabled: boolean;
  system: boolean;
  grantCount: number;
  granted?: boolean;
}

export interface UserBadgeSummary {
  badge: Badge;
  count: number;
  grantedAt: string;
}

export interface BadgeGrant {
  id: number;
  user: User;
  grantedAt: string;
  topic?: Topic;
  post?: Post;
  reason?: string;
}

export interface UserCard {
  user: User;
  badges: UserBadgeSummary[];
  badgeCount: number;
  groups: Group[];
  canMessage: boolean;
  canChat: boolean;
}

export interface PollOption {
  id: string;
  html: string;
  votes: number;
}

export interface Poll {
  id: number;
  postId: number;
  name: string;
  type: "regular" | "multiple";
  status: "open" | "closed";
  results: "always" | "on_vote" | "on_close" | "staff_only";
  public: boolean;
  min: number;
  max: number;
  chartType: string;
  closeAt: string;
  options: PollOption[];
  voters: number;
  userVotes: string[];
  resultsHidden: boolean;
  canClose: boolean;
}

export interface ChatMembership {
  id: number;
  channelId: number;
  userId: number;
  following: boolean;
  muted: boolean;
  lastReadMessageId: number;
}

export interface ChatChannel {
  id: number;
  name: string;
  slug: string;
  description: string;
  channelType: "category" | "direct";
  categoryId: number;
  status: "open" | "read_only" | "closed" | "archived";
  userCount: number;
  messageCount: number;
  lastMessageId: number;
  lastMessageAt: string;
  createdAt: string;
  membership?: ChatMembership;
  unreadCount: number;
  directUsers?: User[];
  canSend: boolean;
  canModerate: boolean;
}

export interface ChatReaction {
  emoji: string;
  count: number;
  reacted: boolean;
  usernames: string[];
}

export interface ChatMessage {
  id: number;
  channelId: number;
  userId: number;
  raw?: string;
  cooked: string;
  excerpt: string;
  inReplyToId: number;
  edited: boolean;
  deleted: boolean;
  createdAt: string;
  updatedAt: string;
  reactions: ChatReaction[];
  inReplyTo?: { id: number; userId: number; excerpt: string };
}

export interface ChatMessageList {
  messages: ChatMessage[];
  users: UserMap;
  morePast: boolean;
  moreFuture: boolean;
}

export interface BusMessage {
  id: number;
  channel: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: any;
}
