import { showcaseService } from "@/services/showcaseService";
import { useQuery } from "@tanstack/react-query";

// Contoh logic useProfile (React Query)
// export const useProfile = () => {
// 	return useQuery({
// 		queryKey: ["profile", "me"],
// 		queryFn: userService.getMe,
// 		retry: false, // Jika gagal (401), jangan coba lagi (anggap guest)
// 		staleTime: 1000 * 60 * 5, // Data dianggap "fresh" selama 5 menit
// 		refetchOnWindowFocus: false, // Jangan refresh saat pindah tab (kecuali perlu real-time)
// 	});
// };

// get detail showcase by slug
export const useDetailShowcase = (slug: string) => {
	return useQuery({
		queryKey: ["showcase", slug], // Cache unik per slug
		queryFn: () => showcaseService.getDetailShowcase(slug),
		enabled: !!slug, // Hanya jalan jika slug ada
		retry: false, // Jangan retry kalau 404
	});
};
