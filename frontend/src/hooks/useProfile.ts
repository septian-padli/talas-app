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
