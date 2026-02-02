export interface SocialLink {
	social: string;
	link: string;
	username?: string | null;
}
// --- User Main Object ---
export interface User {
	id: string;
	username: string;
	email?: string; // only for private profile
	name?: string | null;
	bio?: string | null;
	avatar_url?: string | null;
	job_title?: string | null;
	is_verified?: boolean; // only for private profile
	created_at?: string;
	updated_at?: string;
	social_links?: SocialLink[];
	// stats
	followers_count: number;
	following_count: number;
	is_following?: boolean; // only for public profile (if logged in)
}

export interface UserStats {
	followers_count: number;
	following_count: number;
}

export interface UpdateProfilePayload {
	name?: string;
	bio?: string;
	avatarUrl?: string;
	jobTitle?: string;
	socialLinks?: SocialLink[];
}

export interface FieldError {
	field: string;
	message: string;
}

// --- Response Types ---
export interface ApiResponse<T> {
	code: number;
	success: boolean;
	data: T;
	errors?: FieldError[] | null;
}

export type GetMeResponse = ApiResponse<{ user: User }>;
export type UpdateProfileResponse = ApiResponse<{ user: User }>;
export type GetPublicProfileResponse = ApiResponse<{ user: User }>;
export type ToggleFollowResponse = ApiResponse<{
	is_following: boolean;
	followers_count: number;
}>;
export type UserListItem = User & {
	is_following: boolean;
	followed_at: string;
};
export type CursorPaginationMeta = {
	curr_cursor?: string | null;
	next_cursor?: string | null;
	has_next: boolean;
	limit: number;
};
export type GetFollowersResponse = ApiResponse<{
	users: UserListItem[];
	meta: CursorPaginationMeta;
}>;
export type GetFollowingResponse = GetFollowersResponse;
