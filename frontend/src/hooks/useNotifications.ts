// src/hooks/useNotifications.ts
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationService } from "@/services/notificationService";
import { NotificationItem } from "@/types/notification";

export const useNotifications = () => {
	const queryClient = useQueryClient();

	// 1. Query untuk ambil data
	const query = useQuery({
		queryKey: ["notifications"],
		queryFn: () => notificationService.getAll(),
		// Opsional: Polling setiap 10 detik agar terasa "semi-realtime"
		// refetchInterval: 10000,
		refetchOnWindowFocus: true,
	});

	// 2. Mutation untuk Mark as Read (Optimistic Update)
	const markReadMutation = useMutation({
		mutationFn: notificationService.markAsRead,
		onSuccess: () => {
			// Refresh data setelah berhasil
			queryClient.invalidateQueries({ queryKey: ["notifications"] });
		},
	});

	const markAllReadMutation = useMutation({
		mutationFn: notificationService.markAllRead,
		onSuccess: () => {
			//   console.log("Semua notifikasi telah ditandai dibaca di server.");
		},
	});

	// Hitung jumlah notifikasi yang belum dibaca (unread count)
	const notifications: NotificationItem[] =
		query.data?.data?.notifications || [];
	const unreadCount = notifications.filter(
		(n: NotificationItem) => !n.is_read,
	).length;

	return {
		notifications,
		isLoading: query.isLoading,
		isError: query.isError,
		unreadCount,
		markAsRead: markReadMutation.mutate,
		markAllRead: markAllReadMutation.mutate,
		refetch: query.refetch,
	};
};
