"use client";
import {
    Field,
    FieldDescription,
    FieldGroup,
    FieldLabel,
} from "@/components/ui/field";
import {
    HeartIcon,
    ChatBubbleLeftIcon,
    BookmarkIcon,
    ShareIcon,
} from "@heroicons/react/24/outline";
import {
    HeartIcon as HeartIconSolid,
    BookmarkIcon as BookmarkIconSolid,
} from "@heroicons/react/24/solid";

interface CommentSectionProps {
    prop: string;
}

import React, { useState } from "react";
import { useProfile } from "@/hooks/useProfile";
import CommentForm from "../../../../components/comment/CommentForm";
import CommentItem from "../../../../components/comment/CommentItem";

const CommentSection: React.FC<CommentSectionProps> = () => {
    // get user data yang login
    const {
        data: profileResponse,
        isLoading: loadingUser,
        isError: errorUser,
    } = useProfile();
    const [content, setContent] = useState("");

    return (
        <div className="p-4">
            {/* header */}
            <div className="mb-4">
                <h3 className="font-bold text-xl">3 Komentar</h3>
            </div>

            {/* Form comment */}
            <CommentForm
                value={content}
                onChange={(e) => setContent(e.target.value)}
                onReset={() => setContent("")}
                avatarUrl={profileResponse?.data.user.avatar_url || ""}
            />

            <div className="border-t border-white/10 my-8" />

            {/* list comment */}
            <div className="flex flex-col gap-4">
                {/* comment item */}
                <CommentItem
                    avatarUrl="https://randomuser.me/api/portraits/men/1.jpg"
                    username="dummyuser"
                    jobTitle="jobtitle"
                    timestamp="2 jam lalu"
                    content="Lorem ipsum dolor sit amet consectetur adipisicing elit. Accusamus, autem totam libero aliquid ab nulla quisquam quidem quasi ad non inventore ducimus adipisci, facere hic vel, aliquam ipsam vitae reprehenderit."
                    likes={12}
                    comments={3}
                />
            </div>
        </div>
    );
};

export default CommentSection;
