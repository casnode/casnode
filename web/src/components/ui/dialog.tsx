"use client";

import { Dialog as DialogPrimitive } from "radix-ui";
import * as React from "react";
import { cn } from "@/lib/utils";
import { Icon } from "@/components/icon";

// Discourse's modal: a centered box with a title bar, a body and a footer of buttons.
function Dialog(props: React.ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root data-slot="dialog" {...props} />;
}

function DialogContent({ className, children, title, footer, ...props }: React.ComponentProps<typeof DialogPrimitive.Content> & { title: React.ReactNode; footer?: React.ReactNode }) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="d-modal__backdrop" />
      <DialogPrimitive.Content data-slot="dialog-content" className={cn("d-modal__container", className)} aria-describedby={undefined} {...props}>
        <div className="d-modal__header">
          <DialogPrimitive.Title className="d-modal__title">{title}</DialogPrimitive.Title>
          <DialogPrimitive.Close className="btn btn-flat btn-icon" aria-label="close">
            <Icon name="xmark" />
          </DialogPrimitive.Close>
        </div>
        <div className="d-modal__body">{children}</div>
        {footer ? <div className="d-modal__footer">{footer}</div> : null}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

export { Dialog, DialogContent };
