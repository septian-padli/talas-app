export interface CommentAuthor {
	id: string;
	name: string;
	username: string;
	avatarUrl: string;
}

export interface CommentItem {
	id: string;
	created_at: string;
	updated_at: string;
	deleted_at: string | null;
	showcase_id: string;
	user_id: string;
	body: string;
	parent_id: string | null;
	reply_to: string;
	likes_count: number;
	author: CommentAuthor;
	is_edited: boolean;
}

export interface CommentPagination {
	next_cursor: string;
	has_next: boolean;
}

export interface CommentListData {
	comments: CommentItem[];
	pagination: CommentPagination;
}

export interface CommentResponse {
	code: number;
	success: boolean;
	message: string;
	data: CommentListData;
}
