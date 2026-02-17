"use client";

import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { AxiosError } from "axios";
import { toast } from "sonner";

// Components
import { Button } from "@/components/ui/button";
import {
    Field,
    FieldDescription,
    FieldGroup,
    FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";

// Store & Services
import { useHeaderStore } from "@/store/useHeaderStore";
import { showcaseService } from "@/services/showcaseService"; // Pastikan service ini ada

// Types
import { ApiErrorResponse } from "@/types/error";
import MediaUploader from "./MediaUploader";
import SortableMediaGrid from "./SortableMediaGrid";
import { Category } from "@/types/showcase";
import { categoryService } from "@/services/categoryService";
import { SearchSelect } from "./SearchSelect";


// --- 1. Schema Validation (Zod) ---
// Form schema: media is array of File
const createShowcaseSchema = z.object({
    title: z.string().min(5, { message: "Judul minimal 5 karakter" }).max(200),
    content: z.string().optional(),
    category_id: z.string().uuid({ message: "Kategori wajib dipilih" }),
    tags_input: z.string().optional(),
    media: z.array(z.instanceof(File)).min(1, { message: "Minimal upload 1 media (gambar/video)" })
});
type CreateShowcaseFormValues = z.infer<typeof createShowcaseSchema>;

export default function CreateShowcasePage() {


    // --- 2. Setup Header & Router ---
    const { setTitle } = useHeaderStore();
    const router = useRouter();

    useEffect(() => {
        setTitle("Create Showcase");
        return () => setTitle("Talas App");
    }, [setTitle]);



    // --- 3. State & Form Setup ---
    const [error, setError] = useState<string | null>(null);
    const [isSuccessDelay, setIsSuccessDelay] = useState(false);

    const form = useForm<CreateShowcaseFormValues>({
        resolver: zodResolver(createShowcaseSchema),
        defaultValues: {
            title: "",
            content: "",
            tags_input: "",
            media: []
        }
    });

    const {
        register,
        setValue,
        handleSubmit,
        formState: { errors }
    } = form;

    // Media state for preview and form
    const [mediaFiles, setMediaFiles] = useState<File[]>([]);
    const handleAddMedia = (file: File) => {
        setMediaFiles((prev) => [...prev, file]);
    };
    const handleRemoveMedia = (index: number) => {
        setMediaFiles((prev) => prev.filter((_, i) => i !== index));
    };

    const handleReorderMedia = (newFiles: File[]) => {
        setMediaFiles(newFiles);
    };

    // Sync mediaFiles to react-hook-form's media field for validation and submission
    useEffect(() => {
        setValue("media", mediaFiles, { shouldValidate: true });
    }, [mediaFiles, setValue]);

    // --- 4. Fetch Categories ---

    const { data: categoriesResp } = useQuery({
        queryKey: ['categories'],
        queryFn: categoryService.getCategories
    });
    const categoryItems: { label: string; value: string }[] = categoriesResp?.data.categories.map((cat: Category) => ({
        label: cat.name,
        value: cat.id
    })) || [];




    // --- 5. Mutation (Submit) ---
    const mutation = useMutation({
        mutationFn: async (data: CreateShowcaseFormValues) => {
            // Build FormData for multipart/form-data
            const formData = new FormData();
            formData.append("title", data.title);
            if (data.content) formData.append("content", data.content);
            formData.append("category_id", data.category_id);
            if (data.tags_input) formData.append("tags", data.tags_input);
            // Append all files
            data.media.forEach((file) => {
                formData.append("files", file);
            });

            // Debug: log all FormData entries
            // This will print all key-value pairs, including files
            // Note: File objects will show as File, not the actual content
            console.log("[DEBUG] FormData payload:");
            for (const [key, value] of formData.entries()) {
                if (value instanceof File) {
                    console.log(key, `File(name=${value.name}, type=${value.type}, size=${value.size})`);
                } else {
                    console.log(key, value);
                }
            }

            return await showcaseService.createShowcase(formData);
        },
        onSuccess: (data) => {
            setError(null);
            setIsSuccessDelay(true);
            toast.success("Project berhasil dipublish!", { position: "bottom-right" });
            setTimeout(() => {
                setIsSuccessDelay(false);
                console.log("Create showcase response:", data);
                router.push(`/showcase/${data.data.slug}`);
            }, 2000);
        },
        onError: (err: AxiosError<ApiErrorResponse>) => {
            let msg = "Gagal membuat showcase.";
            if (err?.response?.data?.message) {
                msg = err.response.data.message;
            }
            if (err?.response?.data?.errors) {
                const backendErrors = err.response.data.errors;
                backendErrors.forEach((e) => {
                    toast.error(`${e.field}: ${e.message}`);
                });
            } else {
                setError(msg);
                toast.error(msg);
            }
        },
    });

    const onSubmit = (values: CreateShowcaseFormValues) => {
        setError(null);
        mutation.mutate({ ...values, media: mediaFiles });
    };

    // --- 6. Render UI ---
    return (
        <div className="bg-[#181818] rounded-2xl px-6 py-8 shadow-sm border border-white/5">
            {error && (
                <div className="bg-red-500/80 text-white rounded px-4 py-2 mb-6 text-center text-sm">
                    {error}
                </div>
            )}

            <form onSubmit={handleSubmit(onSubmit)}>
                <FieldGroup>

                    {/* Title Input */}
                    <Field>
                        <FieldLabel htmlFor="title">Judul Project</FieldLabel>
                        <Input
                            id="title"
                            placeholder="Contoh: Redesign Aplikasi Talas"
                            {...register("title")}
                            disabled={mutation.isPending || isSuccessDelay}
                        />
                        {errors.title && (
                            <FieldDescription className="text-rose-400">
                                {errors.title.message}
                            </FieldDescription>
                        )}
                    </Field>

                    {/* Category Select Placeholder - MultiSelect removed. Insert new select here if needed. */}
                    {/* {errors.category_id && (
                        <FieldDescription className="text-rose-400">
                            {errors.category_id.message}
                        </FieldDescription>
                    )} */}

                    <Field>
                        <FieldLabel htmlFor="category_id">Kategori</FieldLabel>
                        <SearchSelect
                            items={categoryItems}
                            placeholder="Pilih kategori..."
                            onSelect={(val) => setValue("category_id", val, { shouldValidate: true })}
                        />
                        {errors.category_id && (
                            <FieldDescription className="text-rose-400">
                                {errors.category_id.message}
                            </FieldDescription>
                        )}
                    </Field>

                    {/* Media Upload Section */}
                    <Field>
                        <FieldLabel>Media Gallery</FieldLabel>

                        {/* Preview List */}
                        {mediaFiles.length > 0 && (
                            <SortableMediaGrid
                                mediaFiles={mediaFiles}
                                onRemove={handleRemoveMedia}
                                onReorder={handleReorderMedia}
                            />
                        )}

                        {/* Uploader Component */}
                        <MediaUploader
                            onFileAdd={handleAddMedia}
                        />
                        {errors.media && form.formState.isSubmitted && (
                            <FieldDescription className="text-rose-400">
                                {errors.media.message}
                            </FieldDescription>
                        )}
                    </Field>

                    {/* Content / Description */}
                    <Field>
                        <FieldLabel htmlFor="content">Deskripsi</FieldLabel>
                        <textarea
                            id="content"
                            {...register("content")}
                            disabled={mutation.isPending || isSuccessDelay}
                            rows={6}
                            placeholder="Ceritakan detail project kamu..."
                            className="w-full bg-[#27272a] border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder:text-zinc-500 focus:outline-none focus:ring-2 focus:ring-brand-500 disabled:opacity-50"
                        />
                        {errors.content && (
                            <FieldDescription className="text-rose-400">
                                {errors.content.message}
                            </FieldDescription>
                        )}
                    </Field>

                    {/* Tags Input */}
                    <Field>
                        <FieldLabel htmlFor="tags">Tags</FieldLabel>
                        <Input
                            id="tags"
                            placeholder="Contoh: React, Golang, UI/UX (pisahkan koma)"
                            {...register("tags_input")}
                            disabled={mutation.isPending || isSuccessDelay}
                        />
                        <FieldDescription>
                            Pisahkan dengan koma untuk banyak tag.
                        </FieldDescription>
                    </Field>

                    {/* Submit Button */}
                    <div className="pt-4">
                        <Button
                            variant={"brand"}
                            type="submit"
                            size={"lg"}
                            disabled={mutation.isPending || isSuccessDelay}
                            className="w-full font-semibold"
                        >
                            {(mutation.isPending || isSuccessDelay) ? "Publishing..." : "Publish Showcase"}
                        </Button>
                    </div>

                </FieldGroup>
            </form>
        </div>
    );
}