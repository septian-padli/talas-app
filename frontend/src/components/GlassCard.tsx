"use client";

export default function GlassCard({ children }: { children: React.ReactNode }) {
    return (
        <div className="max-w-xl w-full rounded-2xl bg-white/10 backdrop-blur-md border border-white/20 shadow-lg p-8">
            {children}
        </div>
    );
}
