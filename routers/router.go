// Copyright 2026 The Casnode Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package routers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/casnode/casnode/controllers"
)

func InitAPI() {
	web.Router("/api/get-site", &controllers.ApiController{}, "GET:GetSite")
	web.Router("/api/signin", &controllers.ApiController{}, "POST:Signin")
	web.Router("/api/dev-signin", &controllers.ApiController{}, "POST:DevSignin")
	web.Router("/api/signout", &controllers.ApiController{}, "POST:Signout")
	web.Router("/api/signup", &controllers.ApiController{}, "POST:Signup")
	web.Router("/api/login", &controllers.ApiController{}, "POST:Login")
	web.Router("/api/forgot-password", &controllers.ApiController{}, "POST:ForgotPassword")
	web.Router("/api/reset-password", &controllers.ApiController{}, "POST:ResetPassword")
	web.Router("/api/change-password", &controllers.ApiController{}, "POST:ChangePassword")
	web.Router("/api/auth/start", &controllers.ApiController{}, "GET:AuthStart")
	web.Router("/api/auth/callback/:provider", &controllers.ApiController{}, "GET:AuthCallback")
	web.Router("/api/get-auth-providers", &controllers.ApiController{}, "GET:GetAuthProviders")
	web.Router("/api/save-auth-provider", &controllers.ApiController{}, "POST:SaveAuthProvider")
	web.Router("/api/delete-auth-provider", &controllers.ApiController{}, "POST:DeleteAuthProvider")
	web.Router("/api/get-account", &controllers.ApiController{}, "GET:GetAccount")

	web.Router("/api/get-topics", &controllers.ApiController{}, "GET:GetTopics")
	web.Router("/api/get-topic", &controllers.ApiController{}, "GET:GetTopic")
	web.Router("/api/get-posts", &controllers.ApiController{}, "GET:GetPosts")
	web.Router("/api/add-topic", &controllers.ApiController{}, "POST:AddTopic")
	web.Router("/api/update-topic", &controllers.ApiController{}, "POST:UpdateTopic")
	web.Router("/api/set-topic-status", &controllers.ApiController{}, "POST:SetTopicStatus")
	web.Router("/api/delete-topic", &controllers.ApiController{}, "POST:DeleteTopic")
	web.Router("/api/recover-topic", &controllers.ApiController{}, "POST:RecoverTopic")
	web.Router("/api/set-topic-notification-level", &controllers.ApiController{}, "POST:SetTopicNotificationLevel")
	web.Router("/api/record-topic-timing", &controllers.ApiController{}, "POST:RecordTopicTiming")
	web.Router("/api/dismiss-topics", &controllers.ApiController{}, "POST:DismissTopics")
	web.Router("/api/invite-to-message", &controllers.ApiController{}, "POST:InviteToMessage")

	web.Router("/api/add-post", &controllers.ApiController{}, "POST:AddPost")
	web.Router("/api/get-post", &controllers.ApiController{}, "GET:GetPost")
	web.Router("/api/update-post", &controllers.ApiController{}, "POST:UpdatePost")
	web.Router("/api/delete-post", &controllers.ApiController{}, "POST:DeletePost")
	web.Router("/api/recover-post", &controllers.ApiController{}, "POST:RecoverPost")
	web.Router("/api/get-post-replies", &controllers.ApiController{}, "GET:GetPostReplies")
	web.Router("/api/get-post-revisions", &controllers.ApiController{}, "GET:GetPostRevisions")
	web.Router("/api/get-post-likers", &controllers.ApiController{}, "GET:GetPostLikers")
	web.Router("/api/like-post", &controllers.ApiController{}, "POST:LikePost")
	web.Router("/api/bookmark-post", &controllers.ApiController{}, "POST:BookmarkPost")
	web.Router("/api/flag-post", &controllers.ApiController{}, "POST:FlagPost")
	web.Router("/api/preview", &controllers.ApiController{}, "POST:Preview")

	web.Router("/api/get-categories", &controllers.ApiController{}, "GET:GetCategories")
	web.Router("/api/add-category", &controllers.ApiController{}, "POST:AddCategory")
	web.Router("/api/update-category", &controllers.ApiController{}, "POST:UpdateCategory")
	web.Router("/api/delete-category", &controllers.ApiController{}, "POST:DeleteCategory")
	web.Router("/api/reorder-categories", &controllers.ApiController{}, "POST:ReorderCategories")
	web.Router("/api/get-tags", &controllers.ApiController{}, "GET:GetTags")
	web.Router("/api/update-tag", &controllers.ApiController{}, "POST:UpdateTag")
	web.Router("/api/delete-tag", &controllers.ApiController{}, "POST:DeleteTag")

	web.Router("/api/get-user", &controllers.ApiController{}, "GET:GetUser")
	web.Router("/api/get-user-activity", &controllers.ApiController{}, "GET:GetUserActivity")
	web.Router("/api/update-profile", &controllers.ApiController{}, "POST:UpdateProfile")
	web.Router("/api/get-directory", &controllers.ApiController{}, "GET:GetDirectory")
	web.Router("/api/search-users", &controllers.ApiController{}, "GET:SearchUsers")
	web.Router("/api/get-notifications", &controllers.ApiController{}, "GET:GetNotifications")
	web.Router("/api/read-notification", &controllers.ApiController{}, "POST:ReadNotification")

	web.Router("/api/get-about", &controllers.ApiController{}, "GET:GetAbout")
	web.Router("/api/get-static-page", &controllers.ApiController{}, "GET:GetStaticPage")
	web.Router("/api/search", &controllers.ApiController{}, "GET:Search")
	web.Router("/api/upload", &controllers.ApiController{}, "POST:Upload")

	web.Router("/api/get-dashboard", &controllers.ApiController{}, "GET:GetDashboard")
	web.Router("/api/get-site-settings", &controllers.ApiController{}, "GET:GetSiteSettings")
	web.Router("/api/update-site-setting", &controllers.ApiController{}, "POST:UpdateSiteSetting")
	web.Router("/api/get-admin-users", &controllers.ApiController{}, "GET:GetAdminUsers")
	web.Router("/api/update-admin-user", &controllers.ApiController{}, "POST:UpdateAdminUser")
	web.Router("/api/get-reviewables", &controllers.ApiController{}, "GET:GetReviewables")
	web.Router("/api/perform-reviewable", &controllers.ApiController{}, "POST:PerformReviewable")
	web.Router("/api/send-test-email", &controllers.ApiController{}, "POST:SendTestEmail")

	web.Router("/api/message-bus-poll", &controllers.ApiController{}, "GET:PollMessageBus")
	web.Router("/api/get-user-card", &controllers.ApiController{}, "GET:GetUserCard")
	web.Router("/api/get-user-preferences", &controllers.ApiController{}, "GET:GetUserPreferences")

	web.Router("/api/get-groups", &controllers.ApiController{}, "GET:GetGroups")
	web.Router("/api/get-group", &controllers.ApiController{}, "GET:GetGroup")
	web.Router("/api/get-group-members", &controllers.ApiController{}, "GET:GetGroupMembers")
	web.Router("/api/get-group-activity", &controllers.ApiController{}, "GET:GetGroupActivity")
	web.Router("/api/get-group-messages", &controllers.ApiController{}, "GET:GetGroupMessages")
	web.Router("/api/add-group", &controllers.ApiController{}, "POST:AddGroup")
	web.Router("/api/update-group", &controllers.ApiController{}, "POST:UpdateGroup")
	web.Router("/api/delete-group", &controllers.ApiController{}, "POST:DeleteGroup")
	web.Router("/api/add-group-members", &controllers.ApiController{}, "POST:AddGroupMembers")
	web.Router("/api/remove-group-member", &controllers.ApiController{}, "POST:RemoveGroupMember")
	web.Router("/api/set-group-owner", &controllers.ApiController{}, "POST:SetGroupOwner")
	web.Router("/api/join-group", &controllers.ApiController{}, "POST:JoinGroup")
	web.Router("/api/request-group-membership", &controllers.ApiController{}, "POST:RequestGroupMembership")
	web.Router("/api/get-group-requests", &controllers.ApiController{}, "GET:GetGroupRequests")
	web.Router("/api/handle-group-request", &controllers.ApiController{}, "POST:HandleGroupRequest")

	web.Router("/api/get-badges", &controllers.ApiController{}, "GET:GetBadges")
	web.Router("/api/get-badge", &controllers.ApiController{}, "GET:GetBadge")
	web.Router("/api/get-user-badges", &controllers.ApiController{}, "GET:GetUserBadges")
	web.Router("/api/add-badge", &controllers.ApiController{}, "POST:AddBadge")
	web.Router("/api/update-badge", &controllers.ApiController{}, "POST:UpdateBadge")
	web.Router("/api/delete-badge", &controllers.ApiController{}, "POST:DeleteBadge")
	web.Router("/api/grant-badge", &controllers.ApiController{}, "POST:GrantBadge")
	web.Router("/api/revoke-badge", &controllers.ApiController{}, "POST:RevokeBadge")

	web.Router("/api/vote-poll", &controllers.ApiController{}, "POST:VotePoll")
	web.Router("/api/remove-poll-vote", &controllers.ApiController{}, "POST:RemovePollVote")
	web.Router("/api/set-poll-status", &controllers.ApiController{}, "POST:SetPollStatus")
	web.Router("/api/get-poll-voters", &controllers.ApiController{}, "GET:GetPollVoters")
	web.Router("/api/move-posts", &controllers.ApiController{}, "POST:MovePosts")
	web.Router("/api/search-topics-for-move", &controllers.ApiController{}, "GET:SearchTopicsForMove")

	web.Router("/api/get-chat-channels", &controllers.ApiController{}, "GET:GetChatChannels")
	web.Router("/api/browse-chat-channels", &controllers.ApiController{}, "GET:BrowseChatChannels")
	web.Router("/api/get-chat-channel", &controllers.ApiController{}, "GET:GetChatChannel")
	web.Router("/api/get-chat-messages", &controllers.ApiController{}, "GET:GetChatMessages")
	web.Router("/api/get-chat-message", &controllers.ApiController{}, "GET:GetChatMessage")
	web.Router("/api/send-chat-message", &controllers.ApiController{}, "POST:SendChatMessage")
	web.Router("/api/edit-chat-message", &controllers.ApiController{}, "POST:EditChatMessage")
	web.Router("/api/delete-chat-message", &controllers.ApiController{}, "POST:DeleteChatMessage")
	web.Router("/api/react-chat-message", &controllers.ApiController{}, "POST:ReactChatMessage")
	web.Router("/api/follow-chat-channel", &controllers.ApiController{}, "POST:FollowChatChannel")
	web.Router("/api/mark-chat-read", &controllers.ApiController{}, "POST:MarkChatRead")
	web.Router("/api/create-chat-channel", &controllers.ApiController{}, "POST:CreateChatChannel")
	web.Router("/api/update-chat-channel", &controllers.ApiController{}, "POST:UpdateChatChannel")
	web.Router("/api/delete-chat-channel", &controllers.ApiController{}, "POST:DeleteChatChannel")
	web.Router("/api/create-direct-channel", &controllers.ApiController{}, "POST:CreateDirectChannel")
}
