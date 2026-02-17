"use client";
import { PostCard } from "@/components/feeds/post-card"
import { useDetailShowcase } from "@/hooks/useShowcase";
import { useHeaderStore } from "@/store/useHeaderStore";
import { useParams } from "next/dist/client/components/navigation";
import { useEffect } from "react";

interface DetilShowcasePageProps {
    prop: string
}
const DetilShowcasePage: React.FC<DetilShowcasePageProps> = () => {
    const { setTitle } = useHeaderStore();
    useEffect(() => {
        setTitle("Showcase Detail");
        return () => setTitle("Talas App");
    }, [setTitle]);

    // get slug from url
    const params = useParams();
    const slug = typeof params.slug === "string" ? params.slug : Array.isArray(params.slug) ? params.slug[0] : "";
    const { data: showcase, isLoading, error } = useDetailShowcase(slug);

    if (isLoading) {
        return <div className="p-8 text-center">Loading...</div>;
    }
    if (error || !showcase) {
        return <div className="p-8 text-center text-red-500">Gagal memuat showcase.</div>;
    }

    return (
        <div className="bg-[#181818] rounded-2xl">
            <div key={showcase.id} className="border-b border-white/10 p-1">
                <PostCard
                    showcase={showcase}
                />
            </div>
        </div>
    );
}

export default DetilShowcasePage