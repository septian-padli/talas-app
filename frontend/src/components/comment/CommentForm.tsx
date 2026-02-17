
import { Field, FieldGroup } from "@/components/ui/field";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import { Button } from "../ui/button";
import React from "react";

interface CommentFormProps {
    value: string;
    onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
    onReset: () => void;
    onSubmit?: (e: React.FormEvent<HTMLFormElement>) => void;
    avatarUrl: string;
}

const CommentForm: React.FC<CommentFormProps> = ({ value, onChange, onReset, onSubmit, avatarUrl }) => {
    return (
        <div className="">
            <form onSubmit={onSubmit || (() => { })}>
                <FieldGroup className="flex flex-row gap-4">
                    <Avatar className="w-8 h-8 md:w-10 md:h-10">
                        <AvatarImage src={avatarUrl} />
                        <AvatarFallback>CN</AvatarFallback>
                    </Avatar>
                    <div className="w-full">
                        <Field>
                            <textarea
                                id="content"
                                rows={3}
                                value={value}
                                onChange={onChange}
                                placeholder="Ceritakan detail project kamu..."
                                className="w-full bg-[#27272a] border border-white/10 rounded-lg p-4 text-sm text-white placeholder:text-zinc-500 focus:outline-none focus:ring-2 focus:ring-brand-500 disabled:opacity-50 resize-none overflow-y-auto"
                            />
                        </Field>
                        {/* Submit Button */}
                        {value.trim() && (
                            <div className="pt-4 flex flex-row justify-end gap-4">
                                <Button
                                    variant={"outline"}
                                    type="reset"
                                    size={"lg"}
                                    className="w-fit font-semibold"
                                    onClick={onReset}
                                >
                                    Batal
                                </Button>
                                <Button
                                    variant={"brand"}
                                    type="submit"
                                    size={"lg"}
                                    className="w-fit font-semibold"
                                >
                                    Kirim
                                </Button>
                            </div>
                        )}
                    </div>
                </FieldGroup>
            </form>
        </div>
    );
};

export default CommentForm