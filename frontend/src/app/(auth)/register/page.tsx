"use client";

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
import { AxiosError } from "axios";
import type { RegisterRequest } from "@/types/auth";
import { ApiErrorResponse } from "@/types/error";
import { toast } from "sonner";

const registerSchema = z.object({
    name: z.string().min(3, { message: "Nama minimal 3 karakter" }),
    username: z.string().min(3, { message: "Username minimal 3 karakter" }),
    email: z.string().email({ message: "Email tidak valid" }),
    password: z.string().min(8, { message: "Password minimal 8 karakter" }),
});

type RegisterFormValues = z.infer<typeof registerSchema>;

export default function RegisterPage() {
    const router = useRouter();
    const [error, setError] = useState<string | null>(null);
    // Track if we are in the success delay state
    const [isSuccessDelay, setIsSuccessDelay] = useState(false);
    const {
        register,
        handleSubmit,
        formState: { errors },
        reset,
    } = useForm<RegisterFormValues>({
        resolver: zodResolver(registerSchema),
    });

    const mutation = useMutation({
        mutationFn: async (data: RegisterRequest) => {
            return await authService.register(data);
        },
        onSuccess: () => {
            setError(null);
            setIsSuccessDelay(true);
            toast.success("Account created successfully! Redirecting to login...", { position: "bottom-right" });
            setTimeout(() => {
                setIsSuccessDelay(false);
                router.push("/login");
            }, 1500);
        },
        onError: (err: AxiosError<ApiErrorResponse>) => {
            let msg = "Registrasi gagal. Silakan cek data Anda.";
            if (err?.response?.data?.message) {
                msg = err.response.data.message;
            }
            setError(msg);
            toast.error(msg);
        },
    });

    const onSubmit = (values: RegisterFormValues) => {
        setError(null);
        mutation.mutate({
            name: values.name,
            username: values.username,
            email: values.email,
            password: values.password,
        });
    };

    return (
        <>
            <h1 className="text-white text-center font-bold text-3xl leading-normal">
                Join Talas and Showcase Your Creations!
            </h1>
            {error && (
                <div className="bg-red-500/80 text-white rounded px-4 py-2 my-4 text-center">
                    {error}
                </div>
            )}
            <form onSubmit={handleSubmit(onSubmit)}>
                <FieldGroup>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-name">Name</FieldLabel>
                        <Input
                            id="fieldgroup-name"
                            placeholder="Jordan Lee"
                            {...register("name")}
                            disabled={mutation.isPending || isSuccessDelay}
                        />
                        {errors.name && (
                            <p className="text-red-400 text-sm mt-1">{errors.name.message}</p>
                        )}
                    </Field>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-username">Username</FieldLabel>
                        <Input
                            id="fieldgroup-username"
                            placeholder="jordanlee"
                            {...register("username")}
                            disabled={mutation.isPending || isSuccessDelay}
                        />
                        {errors.username && (
                            <p className="text-red-400 text-sm mt-1">{errors.username.message}</p>
                        )}
                    </Field>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-email">Email</FieldLabel>
                        <Input
                            id="fieldgroup-email"
                            type="email"
                            placeholder="name@example.com"
                            {...register("email")}
                            disabled={mutation.isPending || isSuccessDelay}
                        />
                        {errors.email && (
                            <p className="text-red-400 text-sm mt-1">{errors.email.message}</p>
                        )}
                        {/* <FieldDescription>
                            We&apos;ll send updates to this address.
                        </FieldDescription> */}
                    </Field>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-password">Password</FieldLabel>
                        <Input
                            id="fieldgroup-password"
                            type="password"
                            placeholder="••••••••"
                            {...register("password")}
                            disabled={mutation.isPending || isSuccessDelay}
                        />
                        {errors.password && (
                            <p className="text-red-400 text-sm mt-1">{errors.password.message}</p>
                        )}
                    </Field>
                    <Field orientation="vertical" className="items-end w-full">
                        <p className="text-sm text-white/80 mb-2 text-right w-full">
                            Sudah punya akun?{' '}
                            <a href="/login" className="text-brand-400 hover:underline font-semibold">Masuk disini</a>
                        </p>
                        <Button variant={"brand"} type="submit" size={"lg"} disabled={mutation.isPending || isSuccessDelay} className="w-full">
                            {(mutation.isPending || isSuccessDelay) ? "Creating account..." : "Submit"}
                        </Button>
                    </Field>
                </FieldGroup>
            </form>
        </>
    );
}
