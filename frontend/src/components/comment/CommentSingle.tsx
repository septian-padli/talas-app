
import ProfileHeader from "../profile/ProfileHeader";
import ActionButton from "../showcase/ActionButton";
import { HeartIcon, ChatBubbleLeftIcon } from "@heroicons/react/24/outline";

interface CommentItemProps {
    avatarUrl: string;
    username: string;
    jobTitle: string;
    timestamp: string;
    content: string;
    likes: number;
    comments: number;
    onLike?: () => void;
    onComment?: () => void;
    liked?: boolean;
}

const CommentSingle: React.FC<CommentItemProps> = ({
    avatarUrl,
    username,
    jobTitle,
    timestamp,
    content,
    likes,
    comments,
    onLike,
    onComment,
    liked = false,
}) => {
    return (
        <div>
            {/* comment header */}
            <ProfileHeader
                avatarUrl={avatarUrl}
                username={username}
                jobTitle={jobTitle}
                timestamp={timestamp}
                profileUrl="#"
            />

            {/* comment content */}
            <div className="pl-11 pt-2">
                <p className="text-sm text-white">{content}</p>
            </div>

            {/* comment action */}
            <div className="flex justify-start items-center mt-2 pl-9 gap-4">
                <ActionButton
                    icon={<HeartIcon className="w-5 h-5" />}
                    label={`${likes} Likes`}
                    onClick={onLike || (() => { })}
                    active={liked}
                />
                <ActionButton
                    icon={<ChatBubbleLeftIcon className="w-5 h-5" />}
                    label={`${comments} Comments`}
                    onClick={onComment || (() => { })}
                />
            </div>
        </div>
    );
};

export default CommentSingle