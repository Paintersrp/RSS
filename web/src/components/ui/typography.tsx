import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "@radix-ui/react-slot";
import { cn } from "@/lib/utils";

const defaultElementByVariant = {
  h1: "h1",
  h2: "h2",
  h3: "h3",
  h4: "h4",
  p: "p",
  blockquote: "blockquote",
  inlineCode: "code",
  lead: "p",
  large: "div",
  small: "small",
  muted: "p",
} as const;

const typographyVariants = cva("", {
  variants: {
    variant: {
      h1: "scroll-m-20 text-center text-4xl font-extrabold tracking-tight text-balance",
      h2: "scroll-m-20 border-b pb-2 text-3xl font-semibold tracking-tight first:mt-0",
      h3: "scroll-m-20 text-2xl font-semibold tracking-tight",
      h4: "scroll-m-20 text-xl font-semibold tracking-tight",
      p: "leading-7 [&:not(:first-child)]:mt-6",
      blockquote: "mt-6 border-l-2 pl-6 italic",
      inlineCode:
        "bg-muted relative rounded px-[0.3rem] py-[0.2rem] font-mono text-sm font-semibold",
      lead: "text-muted-foreground text-xl",
      large: "text-lg font-semibold",
      small: "text-sm leading-none font-medium",
      muted: "text-muted-foreground text-sm",
    },
    center: {
      true: "text-center",
      false: "",
    },
    balance: {
      true: "text-balance",
      false: "",
    },
  },
  defaultVariants: {
    variant: "p",
    center: false,
    balance: false,
  },
});

type Variant = NonNullable<VariantProps<typeof typographyVariants>["variant"]>;

export interface TypographyProps
  extends Omit<React.HTMLAttributes<HTMLElement>, "color">,
    VariantProps<typeof typographyVariants> {
  asChild?: boolean;
}

/**
 * Typography
 * Single component covering headings, paragraphs, blockquotes, code, etc.
 *
 * Usage:
 *  <Typography variant="h1">Title</Typography>
 *  <Typography variant="inlineCode">@radix-ui/react-alert-dialog</Typography>
 *  <Typography as="span" variant="small">Email address</Typography>
 *  <Typography asChild variant="h2"><Link href="/x">Heading Link</Link></Typography>
 */
export function Typography({
  variant = "p",
  asChild,
  center,
  balance,
  className,
  ...props
}: TypographyProps) {
  const Component: any = asChild
    ? Slot
    : (defaultElementByVariant[variant as Variant] ?? "p");

  return (
    <Component
      className={cn(
        typographyVariants({ variant, center, balance }),
        className,
      )}
      {...props}
    />
  );
}
