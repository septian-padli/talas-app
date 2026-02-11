"use client";

import { useEffect, useState } from "react";
import { useForm, useFieldArray } from "react-hook-form";
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
import Image from "next/image";
import MediaUploader from "./MediaUploader";
import { Category } from "@/types/showcase";
import { MultiSelect } from "@/components/ui/multi-select";

// --- 1. Schema Validation (Zod) ---
const createShowcaseSchema = z.object({
    title: z.string().min(5, { message: "Judul minimal 5 karakter" }).max(200),
    content: z.string().optional(),
    category_id: z.string().uuid({ message: "Kategori wajib dipilih" }),
    tags_input: z.string().optional(), // Input string untuk UI (comma separated)
    media: z.array(z.object({
        url: z.string().url(),
        type: z.enum(["image", "video"]),
        order: z.number(),
        alt: z.string().optional()
    })).min(1, { message: "Minimal upload 1 media (gambar/video)" })
});

type CreateShowcaseFormValues = z.infer<typeof createShowcaseSchema>;

export default function CreateShowcasePage() {
    const [selectedValues, setSelectedValues] = useState<string[]>([]);

    const options = [
        { value: "react", label: "React" },
        { value: "vue", label: "Vue.js" },
        { value: "angular", label: "Angular" },
    ];

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
        control,
        handleSubmit,
        formState: { errors }
    } = form;

    // Handle Array Media
    const { fields, append, remove } = useFieldArray({
        control,
        name: "media"
    });

    // --- 4. Fetch Categories ---
    const { data: categoriesData, isLoading: isCatsLoading } = useQuery({
        queryKey: ['categories'],
        queryFn: showcaseService.getCategories
    });

    // --- 5. Mutation (Submit) ---
    const mutation = useMutation({
        mutationFn: async (data: CreateShowcaseFormValues) => {
            // Transform data agar sesuai API Contract
            // API butuh 'tag' (array), kita punya 'tags_input' (string)
            const payload = {
                title: data.title,
                content: data.content,
                category_id: data.category_id,
                media: data.media,
                tag: data.tags_input
                    ? data.tags_input.split(",").map(t => t.trim()).filter(t => t.length > 0)
                    : []
            };
            return await showcaseService.createShowcase(payload);
        },
        onSuccess: () => {
            setError(null);
            setIsSuccessDelay(true);
            toast.success("Project berhasil dipublish!", { position: "bottom-right" });

            // Delay redirect agar user lihat toast
            setTimeout(() => {
                setIsSuccessDelay(false);
                router.push("/showcases/me"); // Redirect ke halaman list sendiri
            }, 1500);
        },
        onError: (err: AxiosError<ApiErrorResponse>) => {
            let msg = "Gagal membuat showcase.";
            if (err?.response?.data?.message) {
                msg = err.response.data.message;
            }
            // Handle validasi spesifik dari backend jika ada
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
        mutation.mutate(values);
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

                    {/* Category Select */}
                    {/* Kita styling manual <select> agar mirip Input karena Input component biasanya type="text" */}
                    <Field>
                        <FieldLabel htmlFor="category">Kategori</FieldLabel>
                        {/* <div className="relative">
                            <select
                                id="category"
                                {...register("category_id")}
                                disabled={isCatsLoading || mutation.isPending || isSuccessDelay}
                                className="w-full bg-[#27272a] border border-white/10 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-brand-500 disabled:opacity-50 appearance-none"
                            >
                                <option value="">-- Pilih Kategori --</option>
                                {categoriesData?.data?.map((cat: Category) => (
                                    <option key={cat.id} value={cat.id}>
                                        {cat.name}
                                    </option>
                                ))}
                            </select>
                            <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-white/50">
                                <svg className="fill-current h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><path d="M9.293 12.95l.707.707L15.657 8l-1.414-1.414L10 10.828 5.757 6.586 4.343 8z" /></svg>
                            </div>
                        </div> */}

                        <MultiSelect
                            options={options}
                            onValueChange={setSelectedValues}
                            defaultValue={selectedValues}
                            responsive={true}
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
                        {fields.length > 0 && (
                            <div className="grid grid-cols-2 gap-4 mb-4">
                                {fields.map((field, index) => (
                                    <div key={field.id} className="relative aspect-video bg-black/50 rounded-lg overflow-hidden border border-white/10 group">
                                        {field.type === 'image' ? (
                                            <Image src={field.url} alt="preview" className="w-full h-full object-cover" />
                                        ) : (
                                            <video src={field.url} className="w-full h-full object-cover" />
                                        )}
                                        <button
                                            type="button"
                                            onClick={() => remove(index)}
                                            className="absolute top-2 right-2 bg-red-500 text-white rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                                        </button>
                                        <div className="absolute bottom-2 left-2 bg-black/70 px-2 py-1 rounded text-xs text-white">
                                            urutan: {index + 1}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}

                        {/* Uploader Component */}
                        <MediaUploader
                            onUploadSuccess={(fileData) => {
                                append({
                                    url: fileData.url,
                                    type: fileData.type,
                                    order: fields.length + 1,
                                    alt: ""
                                });
                            }}
                        />
                        {errors.media && (
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