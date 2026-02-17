import React from "react";
import { DndContext, closestCenter, PointerSensor, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, useSortable, rectSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import Image from "next/image";

interface SortableMediaGridProps {
    mediaFiles: File[];
    onRemove: (index: number) => void;
    onReorder: (newFiles: File[]) => void;
}

function getFileId(file: File, index: number) {
    // id harus unik dan stabil
    return file.name + "-" + file.size + "-" + index;
}

function SortableMediaItem({ file, index, onRemove, id }: { file: File; index: number; onRemove: (index: number) => void; id: string }) {
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
        isDragging,
    } = useSortable({ id });

    const style = {
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.5 : 1,
        zIndex: isDragging ? 10 : 1,
    };

    const url = URL.createObjectURL(file);
    const isImage = file.type.startsWith("image/");

    return (
        <div
            ref={setNodeRef}
            style={style}
            {...attributes}
            {...listeners}
            className="relative aspect-video bg-black/50 rounded-lg overflow-hidden border border-white/10 group"
        >
            {isImage ? (
                <Image src={url} alt="preview" className="w-full h-full object-cover" width={240} height={240} />
            ) : (
                <video src={url} className="w-full h-full object-cover" />
            )}
            <button
                type="button"
                onClick={() => onRemove(index)}
                className="absolute top-2 right-2 bg-red-500 text-white rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity"
            >
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
            <div className="absolute bottom-2 left-2 bg-black/70 px-2 py-1 rounded text-xs text-white">
                urutan: {index + 1}
            </div>
            <div className="absolute bottom-2 right-2 bg-brand-500/80 text-white text-xs px-2 py-1 rounded shadow">Drag</div>
        </div>
    );
}

export default function SortableMediaGrid({ mediaFiles, onRemove, onReorder }: SortableMediaGridProps) {
    const sensors = useSensors(
        useSensor(PointerSensor, {
            activationConstraint: {
                distance: 5,
            },
        })
    );

    const ids = mediaFiles.map(getFileId);

    const handleDragEnd = (event: DragEndEvent) => {
        const { active, over } = event;
        if (!over) return;
        const activeId = String(active.id);
        const overId = String(over.id);
        if (activeId !== overId) {
            const oldIndex = ids.indexOf(activeId);
            const newIndex = ids.indexOf(overId);
            if (oldIndex !== -1 && newIndex !== -1) {
                const newFiles = arrayMove(mediaFiles, oldIndex, newIndex);
                onReorder(newFiles);
            }
        }
    };

    return (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={ids} strategy={rectSortingStrategy}>
                <div className="grid grid-cols-2 gap-4 mb-4">
                    {mediaFiles.map((file, index) => (
                        <SortableMediaItem key={ids[index]} id={ids[index]} file={file} index={index} onRemove={onRemove} />
                    ))}
                </div>
            </SortableContext>
        </DndContext>
    );
}
