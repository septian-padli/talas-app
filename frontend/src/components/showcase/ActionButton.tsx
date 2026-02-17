import { cn } from "@/lib/utils";

interface ActionButtonProps {
    icon: React.ReactNode;
    label: string;
    onClick: (e: React.MouseEvent<HTMLButtonElement>) => void;
    active?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
}
const ActionButton: React.FC<ActionButtonProps> = ({ icon, label, onClick, active = false, disabled = false, fullWidth = false }) => {
    return (
        <button
            onClick={onClick}
            disabled={disabled}
            className={cn(
                "flex items-center gap-1 px-2 py-1 rounded-md transition-all duration-200",
                "hover:bg-gray-700/20 cursor-pointer focus:outline-none active:scale-90",
                "text-xs transform",
                active ? "text-primary" : "text-white/70",
                disabled && "opacity-50 cursor-not-allowed",
                fullWidth && "w-full justify-center"
            )}
            aria-label={label}
        >
            <span className={cn(
                "transition-transform duration-200",
                active ? "text-primary" : "text-white/70"
            )}>
                {icon}
            </span>
            <span className="font-medium">{label}</span>
        </button>
    );
}
export default ActionButton
