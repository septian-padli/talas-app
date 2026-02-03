"use client";

import { useHeaderStore } from "@/store/useHeaderStore";
import { usePathname } from "next/navigation";

export default function HeaderTitle() {
    const { title } = useHeaderStore();
    const pathname = usePathname();
    const isRoot = pathname === "/";
    if (isRoot) return null;

    return (
        <div className="mb-8">
            <h1 className="text-white text-center font-bold text-3xl leading-[225%] font-comfortaa">
                {title}
            </h1>
        </div>
    );
}