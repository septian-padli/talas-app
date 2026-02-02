import GlassCard from "@/components/GlassCard";
import Image from "next/image";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
    return (
        <div className="flex items-center justify-center min-h-screen">
            <div className="relative w-full h-full">
                <div className="absolute inset-0 bg-[url('/img/auth-bg.jpg')] bg-cover bg-center" />
                <div className="relative z-10 flex items-center justify-center min-h-screen px-4">
                    <GlassCard>
                        <div className="flex flex-col gap-6">
                            <div className="flex justify-center">
                                <Image src="/logo/logo-talas/talas-horizontal-white.png" alt="Talas Logo" width={160} height={90} />
                            </div>
                            {children}
                        </div>
                    </GlassCard>
                </div>
            </div>
        </div>
    );
}
