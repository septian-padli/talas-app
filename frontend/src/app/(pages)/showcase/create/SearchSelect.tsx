"use client"

import * as React from "react"
import { Check, ChevronsUpDown } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
    Command,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
} from "@/components/ui/command"
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from "@/components/ui/popover"

// Props biar bisa reusable
interface SearchSelectProps {
    items: { label: string; value: string }[]
    placeholder?: string
    onSelect: (value: string) => void
}

export function SearchSelect({ items, placeholder = "Select item...", onSelect, value }: SearchSelectProps & { value?: string }) {
    const [open, setOpen] = React.useState(false)
    // If value is provided (controlled), use it, else fallback to local state
    const [internalValue, setInternalValue] = React.useState("")
    const selectedValue = value !== undefined ? value : internalValue

    return (
        <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <Button
                    variant="outline"
                    role="combobox"
                    aria-expanded={open}
                    className="w-full justify-between"
                >
                    {selectedValue
                        ? items.find((item) => item.value === selectedValue)?.label
                        : placeholder}
                    <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-full p-0">
                <Command>
                    <CommandInput placeholder={`Search ${placeholder.toLowerCase()}...`} />
                    <CommandList>
                        <CommandEmpty>No item found.</CommandEmpty>
                        <CommandGroup>
                            {items.map((item) => (
                                <CommandItem
                                    key={item.value}
                                    value={item.label}
                                    onSelect={() => {
                                        const newValue = item.value === selectedValue ? "" : item.value
                                        if (value !== undefined) {
                                            onSelect(newValue)
                                        } else {
                                            setInternalValue(newValue)
                                            onSelect(newValue)
                                        }
                                        setOpen(false)
                                    }}
                                >
                                    <Check
                                        className={cn(
                                            "mr-2 h-4 w-4",
                                            selectedValue === item.value ? "opacity-100" : "opacity-0"
                                        )}
                                    />
                                    {item.label}
                                </CommandItem>
                            ))}
                        </CommandGroup>
                    </CommandList>
                </Command>
            </PopoverContent>
        </Popover>
    )
}