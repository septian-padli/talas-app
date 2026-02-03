import { userService } from "@/services/userService";
import { useQuery } from "@tanstack/react-query";

// Contoh logic useProfile (React Query)
export const useProfile = () => {
	return useQuery({
		queryKey: ["profile", "me"],
		queryFn: userService.getMe,
		retry: false, // Jika gagal (401), jangan coba lagi (anggap guest)
		staleTime: 1000 * 60 * 5, // Data dianggap "fresh" selama 5 menit
		refetchOnWindowFocus: false, // Jangan refresh saat pindah tab (kecuali perlu real-time)
	});
};

export const usePublicProfile = (username: string) => {
	return useQuery({
		queryKey: ["profile", username], // Cache unik per username
		queryFn: () => userService.getByUsername(username),
		enabled: !!username, // Hanya jalan jika username ada
		retry: false, // Jangan retry kalau 404
	});
};
