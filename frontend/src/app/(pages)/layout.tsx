import Sidebar from "@/components/sidebar/sidebar";

export default function HomepageLayout({ children }: { children: React.ReactNode }) {
    return (
        <>
            <Sidebar />
            <main className="bg-background min-h-screen lg:pl-72 md:py-6 lg:py-8 xl:py-10">
                {children}
            </main>
        </>
    );
}