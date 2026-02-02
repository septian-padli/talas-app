// src/services/notificationService.ts
import api from "@/lib/axios";

import {
	NotificationQueryParams,
	NotificationResponse,
} from "@/types/notification";

export const notificationService = {
	// Ambil semua notifikasi (support cursor-based pagination)
	getAll: async (cursor?: string, limit = 10, read?: boolean) => {
		// /notifications?cursor=xxx&limit=10&read=true
		const params: NotificationQueryParams = { limit };
		if (cursor) params.cursor = cursor;
		if (typeof read === "boolean") params.read = read;
		const response = await api.get<NotificationResponse>(`/notifications`, {
			params,
		});
		return response.data;
	},

	// Tandai notifikasi sudah dibaca (Mark as Read single)
	markAsRead: async (id: string) => {
		await api.patch(`/notifications/read`, {
			data: { notification_ids: [id] },
		});
	},

	// Tandai banyak notifikasi sudah dibaca (batch)
	markManyRead: async (ids: string[]) => {
		await api.patch(`/notifications/read`, { data: { notification_ids: ids } });
	},

	// Tandai semua sudah dibaca
	markAllRead: async () => {
		await api.patch(`/notifications/read-all`);
	},

	// Get unread notification count (breakdown per type)
	getCount: async () => {
		const response = await api.get(`/notifications/count`);
		return response.data;
	},
};
