import { userMinimal } from "./user";

export interface ShowcaseDetail {
	id: string;
	title: string;
	content?: string;
	slug: string;
	category_id: string;
	category: Category;
	tags?: string[];
	is_edited?: boolean;
	views_count?: number;
	likes_count: number;
	comments_count: number;
	shares_count?: number;
	media?: MediaItem[];
	collaborators?: Array<Collaborator>;
	is_liked?: boolean;
	is_bookmarked?: boolean;
	bookmark_count?: number;
	created_at: string;
	updated_at: string;
	deleted_at?: string | null;
}

export interface Collaborator {
	id: string;
	role: string;
	status: string;
	user: userMinimal;
}

export interface ShowcaseCreateResponse {
	code: number;
	success: boolean;
	data: {
		showcase: ShowcaseDetail;
	};
}
export interface MediaItem {
	id?: string;
	url: string;
	type: "IMAGE" | "VIDEO" | "image" | "video";
	alt?: string;
	position: number;
	created_at?: string;
	updated_at?: string;
	deleted_at?: string | null;
	showcase_id?: string;
}

export interface Category {
	id: string;
	name: string;
	slug: string;
	created_at: string;
	updated_at: string;
}

export interface CategoryPagination {
	has_next: boolean;
	next_cursor: string | null;
}

export interface CategoryListResponse {
	code: number;
	success: boolean;
	message: string;
	data: {
		categories: Category[];
		pagination: CategoryPagination;
	};
}

export interface CreateShowcaseRequest {
	title: string; // required
	category_id: string; // required
	content?: string; // optional
	tag?: string[]; // optional, array of string
	media?: MediaItem[]; // optional, array of MediaItem
}
