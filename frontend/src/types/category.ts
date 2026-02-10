/**
 * Category entity (DTO from backend)
 * createdAt & updatedAt opsional/nullable agar robust jika field tidak ada di response
 */
export interface Category {
	id: string;
	name: string;
	slug: string;
	createdAt?: string | null;
	updatedAt?: string | null;
}
