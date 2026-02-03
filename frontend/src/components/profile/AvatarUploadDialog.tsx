"use client";

import { useState, useRef } from "react";
import AvatarEditor from "react-avatar-editor";
import Dropzone from "react-dropzone";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { userService } from "@/services/userService";
import { toast } from "sonner"; // Atau useToast dari shadcn
import { Loader2, UploadCloud, Image as ImageIcon } from "lucide-react";

interface AvatarUploadDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

const MAX_SIZE_BYTES = 2 * 1024 * 1024;

export default function AvatarUploadDialog({ open, onOpenChange }: AvatarUploadDialogProps) {
    const [image, setImage] = useState<File | null>(null);
    const [scale, setScale] = useState<number>(1);
    const editorRef = useRef<AvatarEditor>(null);
    const queryClient = useQueryClient();

    // Reset state saat modal ditutup
    const handleClose = () => {
        setImage(null);
        setScale(1);
        onOpenChange(false);
    };

    // Mutasi Upload ke Backend
    const { mutate, isPending } = useMutation({
        mutationFn: userService.updateAvatar,
        onSuccess: () => {
            toast.success("Foto profil berhasil diperbarui!");
            // Refresh data user di seluruh aplikasi
            queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
            handleClose();
        },
        onError: () => {
            toast.error("Gagal mengupload foto. Coba lagi.");
        },
    });

    const handleSave = () => {
        if (editorRef.current && image) {
            // Ambil hasil crop sebagai Canvas
            const canvas = editorRef.current.getImageScaledToCanvas();

            // Ubah Canvas jadi Blob (File) lalu upload
            canvas.toBlob((blob) => {
                if (blob) {
                    const formData = new FormData();
                    formData.append("avatar", blob, image.name || "avatar.jpg");
                    mutate(formData);
                }
            }, "image/jpeg");
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Ganti Foto Profil</DialogTitle>
                    <DialogDescription>
                        tekan &apos;Simpan Foto&lsquo; untuk segera mengganti foto profil Anda
                    </DialogDescription>
                </DialogHeader>

                <div className="flex flex-col items-center justify-center gap-6 py-4">
                    {!image ? (
                        // TAMPILAN 1: DROPZONE (Belum ada gambar)
                        <Dropzone
                            maxSize={MAX_SIZE_BYTES}
                            onDrop={(acceptedFiles, fileRejections) => {
                                // Cek jika ada file yang ditolak (misal karena size)
                                if (fileRejections.length > 0) {
                                    const errorType = fileRejections[0].errors[0].code;

                                    if (errorType === "file-too-large") {
                                        toast.error("File terlalu besar! Harus kurang dari 2 MB.");
                                    } else if (errorType === "file-invalid-type") {
                                        toast.error("Format file tidak didukung.");
                                    } else {
                                        toast.error("Gagal memilih file.");
                                    }
                                    return;
                                }

                                // Jika lolos, set image
                                setImage(acceptedFiles[0]);
                            }}
                            noClick={true}
                            multiple={false}
                            accept={{
                                'image/jpeg': [],
                                'image/png': [],
                                'image/webp': []
                            }}
                        >
                            {({ getRootProps, getInputProps, isDragActive, open }) => (
                                <div
                                    {...getRootProps()}
                                    className={`border-2 border-dashed rounded-xl w-full h-64 flex flex-col items-center justify-center cursor-pointer transition-colors ${isDragActive
                                        ? "border-primary bg-primary/10"
                                        : "border-muted-foreground/25 hover:border-primary/50 hover:bg-muted/50"
                                        }`}
                                    onClick={open} // Trigger manual agar area clickable
                                >
                                    <input {...getInputProps()} />
                                    <div className="p-4 rounded-full bg-background border shadow-sm mb-4">
                                        <UploadCloud className="w-8 h-8 text-muted-foreground" />
                                    </div>
                                    <p className="text-sm font-medium text-center px-4">
                                        {isDragActive ? "Lepaskan file di sini" : "Klik atau Drag gambar ke sini"}
                                    </p>
                                    <p className="text-xs text-muted-foreground mt-2">
                                        Support JPG, PNG (Max 2MB)
                                    </p>
                                </div>
                            )}
                        </Dropzone>
                    ) : (
                        // TAMPILAN 2: EDITOR CROP (Gambar sudah dipilih)
                        <div className="flex flex-col items-center w-full animate-in fade-in zoom-in duration-300">
                            <div className="relative border rounded-lg overflow-hidden shadow-sm bg-black/5">
                                <AvatarEditor
                                    ref={editorRef}
                                    image={image}
                                    width={250}
                                    height={250}
                                    border={20}
                                    borderRadius={125} // Membuat mask lingkaran
                                    color={[0, 0, 0, 0.6]} // Warna overlay gelap
                                    scale={scale}
                                    rotate={0}
                                />
                            </div>

                            {/* Slider Zoom */}
                            <div className="w-full max-w-62.5 mt-6 flex items-center gap-3">
                                <ImageIcon className="w-4 h-4 text-muted-foreground" />
                                <Slider
                                    value={[scale]}
                                    min={1}
                                    max={2}
                                    step={0.01}
                                    onValueChange={(val) => setScale(val[0])}
                                    className="flex-1"
                                />
                                <ImageIcon className="w-5 h-5 text-foreground" />
                            </div>
                        </div>
                    )}
                </div>

                <DialogFooter className="flex gap-2 sm:justify-between w-full">
                    {image && (
                        <Button
                            variant="ghost"
                            onClick={() => setImage(null)}
                            disabled={isPending}
                        >
                            Ganti Gambar
                        </Button>
                    )}
                    <div className="flex gap-2 ml-auto">
                        <Button variant="outline" onClick={handleClose} disabled={isPending}>
                            Batal
                        </Button>
                        <Button onClick={handleSave} disabled={!image || isPending}>
                            {isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                            Simpan Foto
                        </Button>
                    </div>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}