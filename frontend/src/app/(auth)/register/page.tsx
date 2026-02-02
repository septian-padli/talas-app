"use client";

import { Button } from "@/components/ui/button";
import {
    Field,
    FieldDescription,
    FieldGroup,
    FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export default function RegisterPage() {
    return (
        <>
            <h1 className="text-white text-center font-bold text-3xl leading-normal">
                Join Talas and Showcase Your Creations!
            </h1>
            <form action="">
                <FieldGroup>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-name">Name</FieldLabel>
                        <Input id="fieldgroup-name" placeholder="Jordan Lee" required />
                    </Field>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-username">Username</FieldLabel>
                        <Input id="fieldgroup-username" placeholder="jordanlee" required />
                    </Field>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-email">Email</FieldLabel>
                        <Input
                            id="fieldgroup-email"
                            type="email"
                            placeholder="name@example.com"
                            required
                        />
                        {/* <FieldDescription>
                            We&apos;ll send updates to this address.
                        </FieldDescription> */}
                    </Field>
                    <Field>
                        <FieldLabel htmlFor="fieldgroup-password">Password</FieldLabel>
                        <Input
                            id="fieldgroup-password"
                            type="password"
                            placeholder="••••••••"
                            required
                        />
                    </Field>
                    <Field orientation="horizontal" className="justify-end">
                        <Button type="reset" variant="outline" size={"lg"}>
                            Reset
                        </Button>
                        <Button variant={"brand"} type="submit" size={"lg"}>Submit</Button>
                    </Field>
                </FieldGroup>
            </form>
        </>
    );
}
