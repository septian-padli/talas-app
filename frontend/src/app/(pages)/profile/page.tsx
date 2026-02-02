"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function ProfileRedirect() {
    const router = useRouter();
    useEffect(() => {
        router.replace("/profile/me");
    }, [router]);
    return null;
}
