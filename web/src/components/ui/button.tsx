import { Slot } from "radix-ui";
import * as React from "react";
import { cn } from "@/lib/utils";

type Variant = "default" | "primary" | "danger" | "flat" | "link";

// Discourse's buttons (.btn, .btn-primary, ...) with shadcn's API.
function Button({
  className,
  variant = "default",
  size,
  asChild = false,
  ...props
}: React.ComponentProps<"button"> & { variant?: Variant; size?: "small" | "icon"; asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "button";
  return (
    <Comp
      data-slot="button"
      className={cn(
        "btn",
        variant === "primary" && "btn-primary",
        variant === "danger" && "btn-danger",
        variant === "flat" && "btn-flat",
        variant === "link" && "btn-link",
        size === "small" && "btn-small",
        size === "icon" && "btn-icon",
        className,
      )}
      {...props}
    />
  );
}

export { Button };
