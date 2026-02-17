import Link from "next/link";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";

interface ProfileHeaderProps {
    avatarUrl: string;
    username: string;
    jobTitle: string;
    timestamp: string;
    profileUrl?: string;
}

const ProfileHeader: React.FC<ProfileHeaderProps> = ({ avatarUrl, username, jobTitle, timestamp, profileUrl = "#" }) => {
    return (
        <div>
            <div className="flex items-center gap-3">
                <Link className="flex gap-3 items-center cursor-pointer" href={profileUrl}>
                    <Avatar className="w-8 h-8 md:w-10 md:h-10">
                        <AvatarImage src={avatarUrl} />
                        <AvatarFallback>{username?.[0]?.toUpperCase() || "U"}</AvatarFallback>
                    </Avatar>
                    <div>
                        <div className="flex flex-row items-center gap-2">
                            <h3 className="font-medium text-foreground text-lg">{username}</h3>
                            <span className="block bg-white/50 w-1.5 h-1.5 rounded-full" />
                            <p className="text-xs text-muted-foreground">{timestamp}</p>
                        </div>
                        <p className="text-muted-foreground">{jobTitle}</p>
                    </div>
                </Link>
                <div className="ml-auto text-xs text-muted-foreground">
                    {timestamp}
                </div>
            </div>
        </div>
    );
};

export default ProfileHeader;