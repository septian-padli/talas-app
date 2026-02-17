import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export const extractUsername = (url: string): string | null => {
	if (!url) return null;

	try {
		// 1. Normalisasi URL (tambahkan https jika belum ada agar URL constructor tidak error)
		const cleanUrl = url.trim();
		const withProtocol = cleanUrl.startsWith("http")
			? cleanUrl
			: `https://${cleanUrl}`;

		// 2. Parsing menggunakan API URL bawaan browser
		const urlObj = new URL(withProtocol);
		const hostname = urlObj.hostname.toLowerCase();
		const pathname = urlObj.pathname;

		// Remove trailing slash agar split lebih bersih
		const cleanPath = pathname.endsWith("/") ? pathname.slice(0, -1) : pathname;
		const segments = cleanPath.split("/").filter(Boolean); // Hapus string kosong

		// --- LOGIC KHUSUS PER PLATFORM ---

		// A. LINKEDIN (Format: /in/username)
		if (hostname.includes("linkedin.com")) {
			// Biasanya ada di segment setelah 'in', misal: /in/septian
			const inIndex = segments.indexOf("in");
			if (inIndex !== -1 && segments[inIndex + 1]) {
				return segments[inIndex + 1];
			}
			// Fallback jika formatnya beda (misal company page)
			return segments[0] || null;
		}

		// B. FACEBOOK (Bisa username atau ID)
		if (hostname.includes("facebook.com")) {
			// Kasus: profile.php?id=1000234
			if (pathname.includes("profile.php")) {
				const id = urlObj.searchParams.get("id");
				return id || null;
			}
			// Kasus: facebook.com/septianpadli
			return segments[0] || null;
		}

		// C. YOUTUBE (Format: /@username atau /c/username atau /user/username)
		if (hostname.includes("youtube.com") || hostname.includes("youtu.be")) {
			// Cari segment yang dimulai dengan @
			const handle = segments.find((s) => s.startsWith("@"));
			if (handle) return handle.replace("@", "");

			// Jika format channel/user biasa, ambil segment terakhir
			return segments[segments.length - 1] || null;
		}

		// D. STANDARD (Instagram, GitHub, X/Twitter, Dribbble)
		// Format umumnya: domain.com/username
		return segments[0] || null;
	} catch {
		// Jika input user bukan URL valid (misal user ngetik "septian" doang)
		// Kita bisa return string aslinya saja (asumsi dia input username langsung)
		return url.trim();
	}
};

// Helper untuk load gambar ke dalam elemen Image
function createImage(url: string): Promise<HTMLImageElement> {
	return new Promise((resolve, reject) => {
		const img = new Image();
		img.addEventListener("load", () => resolve(img));
		img.addEventListener("error", (err) => reject(err));
		img.setAttribute("crossOrigin", "anonymous");
		img.src = url;
	});
}

export async function getCroppedImg(
	imageSrc: string,
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	croppedAreaPixels: any,
): Promise<Blob> {
	const image = await createImage(imageSrc);
	const canvas = document.createElement("canvas");
	const ctx = canvas.getContext("2d");

	canvas.width = croppedAreaPixels.width;
	canvas.height = croppedAreaPixels.height;

	ctx!.drawImage(
		image,
		croppedAreaPixels.x,
		croppedAreaPixels.y,
		croppedAreaPixels.width,
		croppedAreaPixels.height,
		0,
		0,
		croppedAreaPixels.width,
		croppedAreaPixels.height,
	);

	return new Promise<Blob>((resolve) => {
		canvas.toBlob(
			(blob) => {
				if (blob) resolve(blob);
			},
			"image/jpeg",
			1,
		);
	});
}

// function untuk mendapatkan inisial dari nama lengkap. maksimal inisial 2 huruf. jika nama hanya 1 kata, ambil 2 huruf pertama. jika nama lebih dari 1 kata, ambil huruf pertama dari 2 kata pertama
export function getInitials(name: string): string {
	const words = name.trim().split(/\s+/);
	if (words.length === 1) {
		return words[0].substring(0, 2).toUpperCase();
	} else {
		return (words[0][0] + words[1][0]).toUpperCase();
	}
}
