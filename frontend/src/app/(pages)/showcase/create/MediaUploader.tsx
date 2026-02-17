"use client";
import { useDropzone } from "react-dropzone";

interface MediaUploaderProps {
    onFileAdd: (file: File) => void;
}

export default function MediaUploader({ onFileAdd }: MediaUploaderProps) {
    const onDrop = (acceptedFiles: File[]) => {
        for (const file of acceptedFiles) {
            onFileAdd(file);
        }
    };

    const { getRootProps, getInputProps } = useDropzone({
        onDrop,
        accept: { 'image/*': [], 'video/*': [] }
    });

    return (
        <div {...getRootProps()} className="border-2 border-dashed border-gray-300 rounded-lg p-8 text-center cursor-pointer hover:bg-gray-50 transition">
            <input {...getInputProps()} />
            <p className="text-gray-500">Drag & drop gambar/video project di sini, atau klik untuk memilih</p>
        </div>
    );
}