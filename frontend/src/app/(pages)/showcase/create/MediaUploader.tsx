"use client";
import { useDropzone } from "react-dropzone";
import { useState } from "react";
import { showcaseService } from "@/services/showcaseService";

interface MediaUploaderProps {
    onUploadSuccess: (fileData: { url: string; type: "image" | "video" }) => void;
}

export default function MediaUploader({ onUploadSuccess }: MediaUploaderProps) {
    const [isUploading, setIsUploading] = useState(false);

    const onDrop = async (acceptedFiles: File[]) => {
        setIsUploading(true);
        try {
            for (const file of acceptedFiles) {
                const formData = new FormData();
                formData.append("file", file);

                // Upload ke server
                const result = await showcaseService.uploadMedia(formData);

                // Kirim data balik ke parent form
                onUploadSuccess({ url: result.url, type: result.type });
            }
        } catch (error) {
            console.error("Upload failed", error);
            alert("Gagal upload gambar");
        } finally {
            setIsUploading(false);
        }
    };

    const { getRootProps, getInputProps } = useDropzone({
        onDrop,
        accept: { 'image/*': [], 'video/*': [] }
    });

    return (
        <div {...getRootProps()} className="border-2 border-dashed border-gray-300 rounded-lg p-8 text-center cursor-pointer hover:bg-gray-50 transition">
            <input {...getInputProps()} />
            {isUploading ? (
                <p className="text-blue-500">Mengupload...</p>
            ) : (
                <p className="text-gray-500">Drag & drop gambar project di sini, atau klik untuk memilih</p>
            )}
        </div>
    );
}