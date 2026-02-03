"use client"
import { useHeaderStore } from "@/store/useHeaderStore";
import { useEffect } from "react";

interface CreateShowcasePageProps {
    prop: string
}
const CreateShowcasePage: React.FC<CreateShowcasePageProps> = () => {
    const { setTitle } = useHeaderStore();
    useEffect(() => {
        setTitle("Create Showcase");
        return () => setTitle("Talas App");
    }, [setTitle]);
    return (
        <div className="bg-[#181818] rounded-2xl">
            <div className="px-6 py-8">
                create showcase
            </div>
        </div>
    );
}

export default CreateShowcasePage