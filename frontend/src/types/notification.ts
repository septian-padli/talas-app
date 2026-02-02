export enum NotificationType {
	LIKE = "LIKE",
	COMMENT = "COMMENT",
	FOLLOW = "FOLLOW",
	COLLABORATOR_ACCEPTED = "COLLABORATOR_ACCEPTED",
}
export interface NotificationUser {
	id: string;
	username: string;
	avatar_url: string;
}

export interface NotificationEntity {
	id: string;
	type: string;
	title?: string;
	slug?: string;
}

export interface NotificationMeta {
	message_key?: string;
	role?: string;
}

export interface NotificationPreview {
	text?: string;
	id?: string;
}

export interface NotificationItem {
	id: string;
	type: NotificationType | string;
	is_read: boolean;
	created_at: string;
	data: {
		actor?: NotificationUser;
		entity?: NotificationEntity;
		meta?: NotificationMeta;
		preview?: NotificationPreview;
	};
}

export interface NotificationPagination {
	next_cursor: string | null;
	has_next: boolean;
}

export interface NotificationResponse {
	code: number;
	success: boolean;
	data: {
		notifications: NotificationItem[];
		pagination: NotificationPagination;
	};
}

export interface NotificationQueryParams {
	limit: number;
	cursor?: string;
	read?: boolean;
}
