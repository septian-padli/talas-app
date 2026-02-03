
"use client";

import { Button } from "@/components/ui/button";
import {
    Field,
    FieldGroup,
    FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Textarea } from "@/components/ui/textarea";
import { SiInstagram, SiFacebook, SiGithub, SiX, SiDribbble } from "@icons-pack/react-simple-icons";
import { Linkedin } from "lucide-react";
import { useForm, FieldErrorsImpl, FieldValues } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useProfile } from "@/hooks/useProfile";
import { userService } from "@/services/userService";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { extractUsername } from "@/lib/utils";
import { useHeaderStore } from "@/store/useHeaderStore";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";

import { useState } from "react";
import AvatarUploadDialog from "@/components/profile/AvatarUploadDialog";

const socialPlatforms = [
    { key: "instagram", label: "Instagram Link", icon: <SiInstagram /> },
    { key: "facebook", label: "Facebook Link", icon: <SiFacebook /> },
    { key: "github", label: "Github Link", icon: <SiGithub /> },
    { key: "x", label: "X Link", icon: <SiX /> },
    { key: "linkedin", label: "LinkedIn Link", icon: <Linkedin /> },
    { key: "dribbble", label: "Dribbble Link", icon: <SiDribbble /> },
];

const editProfileSchema = z.object({
    name: z.string().min(1, "Name is required"),
    jobTitle: z.string().optional(),
    bio: z.string().optional(),
    avatarUrl: z.string().url("Invalid URL").optional().or(z.literal("")),
    instagram: z.object({ link: z.string().url("Invalid URL").optional().or(z.literal("")) }),
    facebook: z.object({ link: z.string().url("Invalid URL").optional().or(z.literal("")) }),
    github: z.object({ link: z.string().url("Invalid URL").optional().or(z.literal("")) }),
    x: z.object({ link: z.string().url("Invalid URL").optional().or(z.literal("")) }),
    linkedin: z.object({ link: z.string().url("Invalid URL").optional().or(z.literal("")) }),
    dribbble: z.object({ link: z.string().url("Invalid URL").optional().or(z.literal("")) }),
});

type EditProfileForm = z.infer<typeof editProfileSchema>;

// Helper to get nested error message for social fields
function getFieldError<T extends FieldValues>(errors: FieldErrorsImpl<T> | undefined, key: keyof T, field: string): string | undefined {
    const err = errors?.[key];
    if (err && typeof err === "object" && field in err) {
        // @ts-expect-error: safe access for nested error
        return err[field]?.message as string | undefined;
    }
    return undefined;
}

const ProfilePage = () => {
    const { setTitle } = useHeaderStore();
    useEffect(() => {
        setTitle("Edit Profile");
        return () => setTitle("Talas App");
    }, [setTitle]);

    const router = useRouter();
    const { data, isLoading } = useProfile();
    const profile = data?.data?.user;

    const [isUploadOpen, setIsUploadOpen] = useState(false);

    const {
        register,
        handleSubmit,
        setValue,
        formState: { errors, isSubmitting },
    } = useForm<EditProfileForm>({
        resolver: zodResolver(editProfileSchema),
        defaultValues: {
            name: "",
            jobTitle: "",
            bio: "",
            avatarUrl: "",
            instagram: { link: "" },
            facebook: { link: "" },
            github: { link: "" },
            x: { link: "" },
            linkedin: { link: "" },
            dribbble: { link: "" },
        },
    });

    // Prefill form when profile loads
    useEffect(() => {
        if (profile) {
            setValue("name", profile.name || "");
            setValue("jobTitle", profile.job_title || "");
            setValue("bio", profile.bio || "");
            setValue("avatarUrl", profile.avatar_url || "");
            // Map social_links array to flat fields (only link)
            const socials: Record<string, { link: string }> = {};
            (profile.social_links || []).forEach((s) => {
                socials[s.social.toLowerCase()] = { link: s.link };
            });
            socialPlatforms.forEach(({ key }) => {
                setValue(key as keyof EditProfileForm, socials[key.toLowerCase()] || { link: "" });
            });
        }
    }, [profile, setValue]);

    const queryClient = useQueryClient();
    const mutation = useMutation({
        mutationFn: async (data: EditProfileForm) => {
            // Transform flat fields to backend payload, extract username from link
            const socialLinks = socialPlatforms
                .map(({ key }) => {
                    const val = data[key as keyof EditProfileForm] as { link: string };
                    const link = val?.link || "";
                    if (!link) return null;
                    return {
                        social: key.toUpperCase(),
                        link,
                        username: extractUsername(link) || ""
                    };
                })
                .filter((s): s is { social: string; link: string; username: string } => !!s && !!s.link);
            const payload = {
                name: data.name,
                jobTitle: data.jobTitle,
                bio: data.bio,
                avatarUrl: data.avatarUrl,
                socialLinks,
            };
            console.log("Submitting profile update with data:", payload);
            return userService.updateProfile(payload);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
            toast.success("Profile updated!");
            router.push("/profile");
        },
        onError: (err: unknown) => {
            if (err && typeof err === "object" && typeof (err as { message?: unknown }).message === "string") {
                toast.error((err as { message: string }).message || "Failed to update profile");
            } else {
                toast.error("Failed to update profile");
            }
        },
    });

    if (isLoading) {
        return <div className="p-8 text-center">Loading...</div>;
    }

    return (
        <div className="bg-[#181818] rounded-2xl">
            <div className="px-6 py-8">
                <form onSubmit={handleSubmit((data) => mutation.mutate(data))}>
                    <FieldGroup>
                        <div className="grid grid-cols-4 gap-4">
                            <div className="w-full flex flex-col gap-4 col-span-3">
                                {/* Name */}
                                <Field>
                                    <FieldLabel htmlFor="fieldgroup-name">Name</FieldLabel>
                                    <Input id="fieldgroup-name" placeholder="Jordan Lee" {...register("name")} />
                                    {errors.name && <span className="text-red-500 text-xs">{errors.name.message}</span>}
                                </Field>
                                {/* Job Title */}
                                <Field>
                                    <FieldLabel htmlFor="fieldgroup-jobtitle">Job Title</FieldLabel>
                                    <Input id="fieldgroup-jobtitle" placeholder="Frontend Developer" {...register("jobTitle")} />
                                </Field>
                            </div>
                            <div className="col-span-1 flex items-end flex-col gap-2">
                                <div>
                                    <div className="rounded-full overflow-hidden max-w-32 max-h-32 mb-2">
                                        <Avatar>
                                            <AvatarImage src={profile?.avatar_url || undefined} alt={`@${profile?.username}`} />
                                            <AvatarFallback className="text-2xl font-bold">{profile?.name ? profile.name.charAt(0) + profile.name.charAt(1) : "CN"}</AvatarFallback>
                                        </Avatar>
                                    </div>
                                    <Button type="button" variant={"outline"} size={"sm"}
                                        onClick={(e) => {
                                            e.preventDefault();
                                            setIsUploadOpen(true);
                                        }}>
                                        Edit Gambar
                                    </Button>
                                </div>

                            </div>
                        </div>
                        {/* Bio */}
                        <Field>
                            <FieldLabel htmlFor="fieldgroup-bio">Bio</FieldLabel>
                            <Textarea id="fieldgroup-bio" placeholder="Tulis bio singkat..." rows={3} {...register("bio")} />
                        </Field>
                        {/* Social Links */}
                        {socialPlatforms.map(({ key, label, icon }) => (
                            <Field key={key}>
                                <FieldLabel>{label}</FieldLabel>
                                <InputGroup className="w-full">
                                    <InputGroupInput placeholder={label + ' Link'} {...register((key + '.link') as keyof EditProfileForm)} />
                                    <InputGroupAddon>{icon}</InputGroupAddon>
                                </InputGroup>
                                {getFieldError(errors, key as keyof EditProfileForm, 'link') && (
                                    <span className="text-red-500 text-xs">{getFieldError(errors, key as keyof EditProfileForm, 'link')}</span>
                                )}
                            </Field>
                        ))}
                        <Field orientation="vertical" className="items-end w-full">
                            <Button variant={"brand"} type="submit" size={"lg"} className="w-full" disabled={isSubmitting || mutation.isPending}>
                                {mutation.isPending ? "Saving..." : "Edit"}
                            </Button>
                        </Field>
                    </FieldGroup>
                </form>
            </div>

            <AvatarUploadDialog
                open={isUploadOpen}
                onOpenChange={setIsUploadOpen}
            />
        </div>
    );
};

export default ProfilePage;
