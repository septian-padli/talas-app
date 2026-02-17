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
import CommentForm from "../../../../components/comment/CommentForm";
import { useUser } from "@/hooks/useUser";
import { useMutation } from "@tanstack/react-query";
import { commentService } from "@/services/commentService";
import { toast } from "sonner";
import { CommentItem } from "@/types/comment";
import CommentSingle from "@/components/comment/CommentSingle";
import { AxiosError } from "axios";
import { ApiErrorResponse } from "@/types/error";

const CommentSection: React.FC<CommentSectionProps> = ({ prop: showcaseId }) => {
    // get user data yang login
    const {
        data: profileResponse,
        isLoading: loadingUser,
        isError: errorUser,
    } = useUser();
    const [content, setContent] = useState("");
    const [comments, setComments] = useState<CommentItem[]>([]);
    const [isSubmitting, setIsSubmitting] = useState(false);


    // Mutation for posting comment
    const mutation = useMutation({
        mutationFn: async (content: string) => {
            setIsSubmitting(true);
            return await commentService.createComment(showcaseId, content);
        },
        onSuccess: (data) => {
            setIsSubmitting(false);
            setContent("");
            toast.success("Komentar berhasil dikirim!");
            // Optimistic update: push new comment to list
            const user = profileResponse?.data.user;
            const newComment: CommentItem = {
                id: data.data.comment?.id || Math.random().toString(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                deleted_at: null,
                showcase_id: showcaseId,
                user_id: user?.id || "",
                body: data.data.comment?.body || content,
                parent_id: null,
                reply_to: "",
                likes_count: 0,
                author: {
                    id: user?.id || "",
                    name: user?.name || "",
                    username: user?.username || "",
                    avatarUrl: user?.avatarUrl || "",
                },
                is_edited: false,
            };
            setComments((prev) => [newComment, ...prev]);
        },
        onError: (err: AxiosError<ApiErrorResponse>) => {
            setIsSubmitting(false);
            let msg = "Gagal mengirim komentar";
            if (err?.response?.data?.message) {
                msg = err.response.data.message;
            } else if (typeof err?.message === "string") {
                msg = err.message;
            }
            toast.error(msg);
        },
    });

    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        if (!content.trim()) return;
        mutation.mutate(content);
    };

    return (
        <div className="p-4">
            {/* header */}
            <div className="mb-4">
                <h3 className="font-bold text-xl">Komentar</h3>
            </div>

            {/* Form comment */}
            <CommentForm
                value={content}
                onChange={(e) => setContent(e.target.value)}
                onReset={() => setContent("")}
                loadingUser={loadingUser || isSubmitting}
                user={profileResponse?.data.user}
                onSubmit={handleSubmit}
            />

            <div className="border-t border-white/10 my-8" />

            {/* list comment */}
            <div className="flex flex-col gap-4">
                {comments.length === 0 && (
                    <div className="text-zinc-400 text-sm">Belum ada komentar.</div>
                )}
                {comments.map((comment) => (
                    <CommentSingle
                        key={comment.id}
                        avatarUrl={comment.author.avatarUrl}
                        username={comment.author.username}
                        jobTitle={"comment.author.jobTitle"}
                        timestamp={comment.created_at}
                        content={comment.body}
                        likes={comment.likes_count}
                        comments={1}
                    />
                ))}
            </div>
        </div>
    );
};

export default CommentSection;
