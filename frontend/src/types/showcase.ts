export interface ShowcaseDetail {
	id: string;
	title: string;
	content?: string;
	slug: string;
	category: Category;
	tag?: string[];
	media?: MediaItem[];
	owner: {
		id: string;
		username: string;
		full_name: string;
		avatar_url: string;
	};
	collaborators?: Array<{
		id: string;
		username: string;
		full_name: string;
		avatar_url: string;
		role: string;
		status: string;
	}>;
	like_count: number;
	comment_count: number;
	bookmark_count: number;
	is_liked?: boolean;
	is_bookmarked?: boolean;
	created_at: string;
	updated_at: string;
}

export interface ShowcaseCreateResponse {
	code: number;
	success: boolean;
	data: {
		showcase: ShowcaseDetail;
	};
}
export interface MediaItem {
	url: string;
	type: "image" | "video";
	alt?: string;
	order: number;
}

export interface Category {
	id: string;
	name: string;
	slug: string;
}

export interface CreateShowcaseRequest {
	title: string; // required
	category_id: string; // required
	content?: string; // optional
	tag?: string[]; // optional, array of string
	media?: MediaItem[]; // optional, array of MediaItem
}
