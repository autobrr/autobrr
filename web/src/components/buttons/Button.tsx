/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ButtonHTMLAttributes, forwardRef } from "react";
import classNames from "classnames";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "danger";
  size?: "sm" | "md" | "lg";
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "primary", size = "md", children, ...props }, ref) => {
    const baseClasses = "inline-flex items-center justify-center font-medium transition rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2";
    
    const variantClasses = {
      primary: "text-white bg-blue-600 hover:bg-blue-700 focus:ring-blue-500",
      secondary: "text-gray-700 bg-gray-100 hover:bg-gray-200 focus:ring-gray-500 dark:text-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600",
      danger: "text-white bg-red-600 hover:bg-red-700 focus:ring-red-500"
    };

    const sizeClasses = {
      sm: "px-2 py-1 text-sm",
      md: "px-4 py-2 text-sm",
      lg: "px-6 py-3 text-base"
    };

    return (
      <button
        ref={ref}
        className={classNames(
          baseClasses,
          variantClasses[variant],
          sizeClasses[size],
          className
        )}
        {...props}
      >
        {children}
      </button>
    );
  }
);
