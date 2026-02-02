"use client";
import Image from "next/image";
import { LogoutButton } from "../auth/LogoutButton";
import SideNav from "./sidenav";
import { usePathname } from "next/navigation";

// interface SidebarProps {
//     activeItem?: string;
// }

const pathToLabel: Record<string, string> = {
    "/": "Home",
    "/bookmark": "Bookmark",
    "/search": "Search",
    "/notification": "Notification",
    "/profile": "Profile",
};

const Sidebar: React.FC = () => {
    const pathname = usePathname();
    // Find the best match for the current path
    const activeItem = pathToLabel[
        Object.keys(pathToLabel).find((key) => pathname === key || pathname.startsWith(key + "/")) || "/"
    ] || "Home";
    return (
        <>
            {/* Desktop Sidebar */}
            <aside
                className={
                    "hidden md:flex h-screen bg-background fixed left-0 top-0 flex-col justify-between py-7 w-17.5 md:items-center px-0 md:w-67.5 md:px-4"
                }
            >
                <div className={"flex justify-center"}>
                    <Image
                        src="/logo/logo-talas/talas-horizontal-white.png"
                        alt="Talas Logo"
                        width={120}
                        height={120}
                        className="object-contain"
                    />
                </div>
                <SideNav activeItem={activeItem} />
                <LogoutButton variant={"outline"} size={"lg"} className="w-full" />
            </aside>

            {/* mobile header nav */}
            <nav className="bg-background flex py-4 px-4 justify-start items-center md:hidden">
                <Image
                    src="/logo/logo-talas/talas-horizontal-white.png"
                    alt="Talas Logo"
                    width={90}
                    height={90}
                    className="object-contain"
                />
            </nav>

            {/* Mobile Bottom Nav */}
            <nav className="fixed bottom-0 left-0 right-0 z-50 bg-background border-t border-white/10 flex md:hidden justify-evenly items-center h-16">
                <SideNav activeItem={activeItem} orientation="horizontal" />
            </nav>
        </>
    )
}

export default Sidebar