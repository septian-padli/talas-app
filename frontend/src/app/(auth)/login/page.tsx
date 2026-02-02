"use client";

import { AxiosError } from "axios";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { authService } from "@/services/authService";
import { Button } from "@/components/ui/button";
import {
    Field,
    FieldDescription,
    FieldGroup,
    FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ApiErrorResponse } from "@/types/error";
import { toast } from "sonner";

const loginSchema = z.object({
    email: z.string().email({ message: "Email tidak valid" }),
    password: z.string().min(1, { message: "Password wajib diisi" }),
});

type LoginFormValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
    const router = useRouter();
    // Remove local error state, use toast instead
    const {
        register,
        handleSubmit,
        formState: { errors },
        reset,
    } = useForm<LoginFormValues>({
        resolver: zodResolver(loginSchema),
        mode: "onSubmit",
    });

    const mutation = useMutation({
        mutationFn: async (data: LoginFormValues) => {
            return await authService.login(data);
        },
        onSuccess: (data) => {
            toast.success("Login berhasil!", { position: "bottom-right" });
            if (data?.data?.user) {
                if (typeof window !== "undefined") {
                    localStorage.setItem("user_data", JSON.stringify(data.data.user));
                }
            }
            router.replace("/");
        },
        onError: (err: AxiosError<ApiErrorResponse>) => {
            let msg = "Login gagal. Silakan cek email dan password Anda.";
            if (err?.response?.data?.message) {
                msg = err.response.data.message;
            }
            toast.error(msg, { position: "bottom-right" });

        },
    });

    const onSubmit = (values: LoginFormValues) => {
        mutation.mutate(values);
    };

    return (
        <>
            <h1 className="text-white text-center font-bold text-3xl leading-normal">
                Join Talas and Showcase Your Creations!
            </h1>
            {/* Error feedback is now handled by toast */}
            <form onSubmit={handleSubmit(onSubmit)}>
                <FieldGroup>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-email">Email</FieldLabel>
                        <Input
                            id="fieldgroup-email"
                            type="email"
                            placeholder="name@example.com"
                            {...register("email")}
                            disabled={mutation.isPending}
                        />
                        {errors.email ? (
                            <FieldDescription className="text-rose-400">
                                {errors.email.message}
                            </FieldDescription>
                        ) : null}
                    </Field>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-password">Password</FieldLabel>
                        <Input
                            id="fieldgroup-password"
                            type="password"
                            placeholder="••••••••"
                            {...register("password")}
                            disabled={mutation.isPending}
                        />
                        {errors.password ? (
                            <FieldDescription className="text-rose-400">
                                {errors.password.message}
                            </FieldDescription>
                        ) : null}
                    </Field>
                    <Field orientation="vertical" className="items-end w-full">
                        <p className="text-sm text-white/80 mb-2 text-right w-full">
                            Belum punya akun?{' '}
                            <a href="/register" className="text-brand-400 hover:underline font-semibold">Register sekarang</a>
                        </p>
                        <Button variant={"brand"} type="submit" size={"lg"} disabled={mutation.isPending} className="w-full">
                            {mutation.isPending ? "Loading..." : "Submit"}
                        </Button>
                    </Field>
                </FieldGroup>
            </form>
        </>
    );
}
