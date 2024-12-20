/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { classNames } from "@utils";
import { SVGProps } from "react";

type SectionProps = {
  title: string;
  titleElement?: React.ReactNode;
  description: string | React.ReactNode;
  rightSide?: React.ReactNode;
  children?: React.ReactNode;
  noLeftPadding?: boolean;
};

export const Section = ({
  title,
  titleElement,
  description,
  rightSide,
  children,
  noLeftPadding = false,
}: SectionProps) => (
  <div
    className={classNames(
      "pb-6 px-4 lg:col-span-9",
      noLeftPadding ? 'pl-0' : '',
    )}
  >
    <div
      className={classNames(
        "mt-6 mb-4",
        rightSide
          ? "flex justify-between items-start flex-wrap sm:flex-nowrap gap-2"
          : ""
      )}
    >
      <div className="sm:px-2">
        {titleElement ?? <h2 className="text-lg leading-4 font-bold text-gray-900 dark:text-white">{title}</h2>}
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{description}</p>
      </div>
      <div className="flex-shrink-0">
        {rightSide ?? null}
      </div>
    </div>
    {children}
  </div>
);

interface RowItemProps {
  label: string;
  value?: string | React.ReactNode;
  title?: string;
  emptyText?: string;
  rightSide?: React.ReactNode;
  className?: string;
}

export const RowItem = ({
  label,
  value,
  title,
  emptyText,
  rightSide,
  className = "sm:col-span-3"
}: RowItemProps) => (
  <div className="p-4 sm:px-6 sm:grid sm:grid-cols-4 sm:gap-4">
    <div className="font-medium text-gray-900 dark:text-white text-sm self-center" title={title}>
      {label}
    </div>
    <div
      className={classNames(
        className,
        "mt-1 text-gray-900 dark:text-gray-300 text-sm break-all sm:mt-0"
      )}
    >
      {value
        ? (
          <>
            {typeof (value) === "string" ? (
              <span className="px-1.5 py-1 bg-gray-200 dark:bg-gray-700 rounded shadow text-ellipsis leading-7">
                {value}
              </span>
            ) : value}
            {rightSide ?? null}
          </>
        )
        : (emptyText ?? null)
      }
    </div>
  </div>
);

const commonSVGProps: SVGProps<SVGSVGElement> = {
  clipRule: "evenodd", fill: "currentColor", fillRule: "evenodd", xmlns: "http://www.w3.org/2000/svg",
  className: "mr-2 h-5"
};

export const DiscordIcon = () => (
  <svg {...commonSVGProps} viewBox="0 0 50 50">
    <path strokeWidth="1" stroke="currentColor" d="M 18.90625 7 C 18.90625 7 12.539063 7.4375 8.375 10.78125 C 8.355469 10.789063 8.332031 10.800781 8.3125 10.8125 C 7.589844 11.480469 7.046875 12.515625 6.375 14 C 5.703125 15.484375 4.992188 17.394531 4.34375 19.53125 C 3.050781 23.808594 2 29.058594 2 34 C 1.996094 34.175781 2.039063 34.347656 2.125 34.5 C 3.585938 37.066406 6.273438 38.617188 8.78125 39.59375 C 11.289063 40.570313 13.605469 40.960938 14.78125 41 C 15.113281 41.011719 15.429688 40.859375 15.625 40.59375 L 18.0625 37.21875 C 20.027344 37.683594 22.332031 38 25 38 C 27.667969 38 29.972656 37.683594 31.9375 37.21875 L 34.375 40.59375 C 34.570313 40.859375 34.886719 41.011719 35.21875 41 C 36.394531 40.960938 38.710938 40.570313 41.21875 39.59375 C 43.726563 38.617188 46.414063 37.066406 47.875 34.5 C 47.960938 34.347656 48.003906 34.175781 48 34 C 48 29.058594 46.949219 23.808594 45.65625 19.53125 C 45.007813 17.394531 44.296875 15.484375 43.625 14 C 42.953125 12.515625 42.410156 11.480469 41.6875 10.8125 C 41.667969 10.800781 41.644531 10.789063 41.625 10.78125 C 37.460938 7.4375 31.09375 7 31.09375 7 C 31.019531 6.992188 30.949219 6.992188 30.875 7 C 30.527344 7.046875 30.234375 7.273438 30.09375 7.59375 C 30.09375 7.59375 29.753906 8.339844 29.53125 9.40625 C 27.582031 9.09375 25.941406 9 25 9 C 24.058594 9 22.417969 9.09375 20.46875 9.40625 C 20.246094 8.339844 19.90625 7.59375 19.90625 7.59375 C 19.734375 7.203125 19.332031 6.964844 18.90625 7 Z M 18.28125 9.15625 C 18.355469 9.359375 18.40625 9.550781 18.46875 9.78125 C 16.214844 10.304688 13.746094 11.160156 11.4375 12.59375 C 11.074219 12.746094 10.835938 13.097656 10.824219 13.492188 C 10.816406 13.882813 11.039063 14.246094 11.390625 14.417969 C 11.746094 14.585938 12.167969 14.535156 12.46875 14.28125 C 17.101563 11.410156 22.996094 11 25 11 C 27.003906 11 32.898438 11.410156 37.53125 14.28125 C 37.832031 14.535156 38.253906 14.585938 38.609375 14.417969 C 38.960938 14.246094 39.183594 13.882813 39.175781 13.492188 C 39.164063 13.097656 38.925781 12.746094 38.5625 12.59375 C 36.253906 11.160156 33.785156 10.304688 31.53125 9.78125 C 31.59375 9.550781 31.644531 9.359375 31.71875 9.15625 C 32.859375 9.296875 37.292969 9.894531 40.3125 12.28125 C 40.507813 12.460938 41.1875 13.460938 41.8125 14.84375 C 42.4375 16.226563 43.09375 18.027344 43.71875 20.09375 C 44.9375 24.125 45.921875 29.097656 45.96875 33.65625 C 44.832031 35.496094 42.699219 36.863281 40.5 37.71875 C 38.5 38.496094 36.632813 38.84375 35.65625 38.9375 L 33.96875 36.65625 C 34.828125 36.378906 35.601563 36.078125 36.28125 35.78125 C 38.804688 34.671875 40.15625 33.5 40.15625 33.5 C 40.570313 33.128906 40.605469 32.492188 40.234375 32.078125 C 39.863281 31.664063 39.226563 31.628906 38.8125 32 C 38.8125 32 37.765625 32.957031 35.46875 33.96875 C 34.625 34.339844 33.601563 34.707031 32.4375 35.03125 C 32.167969 35 31.898438 35.078125 31.6875 35.25 C 29.824219 35.703125 27.609375 36 25 36 C 22.371094 36 20.152344 35.675781 18.28125 35.21875 C 18.070313 35.078125 17.8125 35.019531 17.5625 35.0625 C 16.394531 34.738281 15.378906 34.339844 14.53125 33.96875 C 12.234375 32.957031 11.1875 32 11.1875 32 C 10.960938 31.789063 10.648438 31.699219 10.34375 31.75 C 9.957031 31.808594 9.636719 32.085938 9.53125 32.464844 C 9.421875 32.839844 9.546875 33.246094 9.84375 33.5 C 9.84375 33.5 11.195313 34.671875 13.71875 35.78125 C 14.398438 36.078125 15.171875 36.378906 16.03125 36.65625 L 14.34375 38.9375 C 13.367188 38.84375 11.5 38.496094 9.5 37.71875 C 7.300781 36.863281 5.167969 35.496094 4.03125 33.65625 C 4.078125 29.097656 5.0625 24.125 6.28125 20.09375 C 6.90625 18.027344 7.5625 16.226563 8.1875 14.84375 C 8.8125 13.460938 9.492188 12.460938 9.6875 12.28125 C 12.707031 9.894531 17.140625 9.296875 18.28125 9.15625 Z M 18.5 21 C 15.949219 21 14 23.316406 14 26 C 14 28.683594 15.949219 31 18.5 31 C 21.050781 31 23 28.683594 23 26 C 23 23.316406 21.050781 21 18.5 21 Z M 31.5 21 C 28.949219 21 27 23.316406 27 26 C 27 28.683594 28.949219 31 31.5 31 C 34.050781 31 36 28.683594 36 26 C 36 23.316406 34.050781 21 31.5 21 Z M 18.5 23 C 19.816406 23 21 24.265625 21 26 C 21 27.734375 19.816406 29 18.5 29 C 17.183594 29 16 27.734375 16 26 C 16 24.265625 17.183594 23 18.5 23 Z M 31.5 23 C 32.816406 23 34 24.265625 34 26 C 34 27.734375 32.816406 29 31.5 29 C 30.183594 29 29 27.734375 29 26 C 29 24.265625 30.183594 23 31.5 23 Z" />
  </svg>
);

export const NotifiarrIcon = () => (
  <svg {...commonSVGProps} viewBox="0 0 144 144">
    <path d="m68.4 2.3c-1.7 4.2 0.5 7.6 5.6 8.6 1.4 0.2 2.9 1.3 3.5 2.3 0.5 1 3.4 5.4 6.2 9.8 7.1 10.7 7.5 10.1-7.5 9.6-15.5-0.5-31.6 1.8-47.4 6.9-10 3.3-10.6 3.4-13.6 1.9-5.7-2.7-10.2-0.1-10.2 5.8 0 6.1 5.5 9.3 10.8 6.1 4-2.4 29.6 14.9 40.1 27 13.8 15.9 21.7 42.3 14.2 47.4-6.5 4.4-2.9 14.3 4.8 13.1 2-0.3 4.2-0.6 4.9-0.6 1.9 0 3.3-4.4 2.2-7.2-0.8-2-0.2-3.5 4.4-10.2 8.2-12 14.6-27.9 14.6-36.1 0-5.9-3.5-4-4.4 2.4-2.9 18.7-19.9 45.2-20.3 31.6-0.8-25.8-19.3-50.6-49.9-66.9-9.6-5.1-8.2-7.9 5.7-12.2 17.3-5.2 42.4-7.8 54.9-5.6 7.6 1.3 15.5 31.8 9.3 36-4.2 2.8 0.2 11.4 4.7 9.1 1.2-0.7 2.4-0.9 2.7-0.6 3 2.8 3.8-3.4 0.9-6.8-1.9-2.2-2.4-4.1-2.9-11.2-0.5-7.1-2.5-16-5.4-23-0.3-0.9 0.2-1.4 1.4-1.4 6.6 0 33.3 11.5 33.3 14.3 0 4.6 10.5 6.6 11.6 2.3 1.2-4.4-3.6-8.6-8.7-7.6-2.2 0.4-5.3-0.6-12.8-4.1-5.5-2.6-13.9-5.6-18.6-6.8-8.6-2.2-8.6-2.2-11.5-7.6-1.6-3-4.9-8.2-7.3-11.6-3.2-4.4-4.3-6.9-4-8.5 1-5.2-9.4-10.9-11.3-6.2z" />
    <path d="m67.9 5.6c0.6 3.4 1.8 4.4 6.1 5.3 1.4 0.2 2.9 1.3 3.5 2.3 0.5 1 3.4 5.4 6.2 9.8 2.9 4.3 5.3 8.6 5.3 9.4 0 0.8-1.8-1.4-3.9-4.9-5.9-9.7-9-13.2-11.6-13.2-5.8 0-9.4-4.6-7.3-9.2 1.1-2.7 1.1-2.7 1.7 0.5zm-4.7-1.7c0.5-0.1 0.8 1.4 0.8 3.2 0 1.8 0.2 3.8 0.5 4.5 0.4 1-1.2 1.6-5.6 2.4-12.9 2.4-27.1 10.8-35.6 21.3-3.8 4.7-4.9 5.5-6.4 4.7-1-0.5-2.9-0.9-4.3-0.9-3.4 0-3.3-0.2 1.8-7 9.8-13.2 26-23.6 42.1-26.9 3.3-0.7 6.3-1.3 6.7-1.3zm21.3 0.5c20.4 3.4 38.8 15.6 50 33.3 4.2 6.7 4.2 6.7 0.7 7.4-1.5 0.3-3.4 0.8-4.3 1.1-1.2 0.5-2.5-0.8-4.9-4.9-7.1-12.1-24.6-24.3-38.9-27.1-5.9-1.2-5.9-1.2-5.7-5.7 0.1-4.5 0.1-4.5 3.1-4.1zm3 31.7c5.5 0.9 9.6 11.5 10.1 26.4 0.3 9.5 0.3 9.5-0.6 0.4-1.8-18.8-4.9-24.5-13.8-25.3-16.8-1.5-39.2 1.4-55.5 7.2-3.2 1.2-6.1 1.9-6.4 1.6-1.5-1.5 18.4-7.6 31.7-9.7 8.2-1.3 28.1-1.7 34.5-0.6zm10.2 2c6.6 0 33.3 11.5 33.3 14.3 0 2.6 3.6 5.1 6.2 4.5 3.4-0.8 5.7 0.8 3 2.1-5.2 2.7-11.2 0.2-11.2-4.6 0-3.1-9.7-8.1-23.7-12.4-4.6-1.4-8.6-2.9-9-3.2-0.3-0.4 0.3-0.7 1.4-0.7zm-92.5 8.2c-0.6 6.8 4.9 10.4 10.6 7 1.5-0.9 3.4-0.2 10.4 3.8 29.3 16.5 44.4 36.7 46.1 61.7 0.6 7.6 0.6 7.6-0.4 1.4-4.5-28-17.3-44.9-46.2-61.3-6.7-3.8-6.7-3.8-10.3-2.2-8.2 3.7-16-3.6-12.1-11.5 2.1-4.2 2.3-4.1 1.9 1.1zm-0.3 11.3c0.7 0.8 2.2 1.5 3.3 1.7 2 0.3 2.1 1 2.3 12.7 0.5 28.8 19.3 51.9 46.9 57.8 5.2 1.1 5.2 1.1 5.2 5.7 0 5.2 0.9 4.9-9.1 2.6-34-7.9-57.8-43.9-51.9-78.3 0.7-3.8 1.5-4.3 3.3-2.2zm135.6 35.6c-7.5 22.6-35.3 46-54.7 46-5.8 0-2.1-8.3 4.6-10.2 27.2-7.9 45.9-33 44.4-59.6-0.5-7.4-0.5-7.4 2.3-7.4 1.5 0 3.1-0.4 3.4-1 3.6-5.5 3.6 21.3 0 32.2zm-45.5-16.6c0 2.7 3.4 5.9 5.1 4.9 3-1.8 6.2-1.4 4.3 0.6-1.5 1.6-4.6 2.1-8.5 1.2-2.1-0.5-4.1-4.8-3.4-7.1 0.9-2.5 2.5-2.3 2.5 0.4zm-26.3 51.2c0.4 0 0.2 0.5-0.5 1.1-5 4.8 0.5 12.7 8.1 11.8 5.2-0.6 5.2-0.6 2.8 1.3-7.6 6.3-18.5-2.9-13.2-11.1 1.2-1.7 2.4-3.1 2.8-3.1z" />
  </svg>
);

export const TelegramIcon = () => (
  <svg {...commonSVGProps} viewBox="0 0 50 50">
    <path d="M 25 2 C 12.309288 2 2 12.309297 2 25 C 2 37.690703 12.309288 48 25 48 C 37.690712 48 48 37.690703 48 25 C 48 12.309297 37.690712 2 25 2 z M 25 4 C 36.609833 4 46 13.390175 46 25 C 46 36.609825 36.609833 46 25 46 C 13.390167 46 4 36.609825 4 25 C 4 13.390175 13.390167 4 25 4 z M 34.087891 14.035156 C 33.403891 14.035156 32.635328 14.193578 31.736328 14.517578 C 30.340328 15.020578 13.920734 21.992156 12.052734 22.785156 C 10.984734 23.239156 8.9960938 24.083656 8.9960938 26.097656 C 8.9960938 27.432656 9.7783594 28.3875 11.318359 28.9375 C 12.146359 29.2325 14.112906 29.828578 15.253906 30.142578 C 15.737906 30.275578 16.25225 30.34375 16.78125 30.34375 C 17.81625 30.34375 18.857828 30.085859 19.673828 29.630859 C 19.666828 29.798859 19.671406 29.968672 19.691406 30.138672 C 19.814406 31.188672 20.461875 32.17625 21.421875 32.78125 C 22.049875 33.17725 27.179312 36.614156 27.945312 37.160156 C 29.021313 37.929156 30.210813 38.335938 31.382812 38.335938 C 33.622813 38.335938 34.374328 36.023109 34.736328 34.912109 C 35.261328 33.299109 37.227219 20.182141 37.449219 17.869141 C 37.600219 16.284141 36.939641 14.978953 35.681641 14.376953 C 35.210641 14.149953 34.672891 14.035156 34.087891 14.035156 z M 34.087891 16.035156 C 34.362891 16.035156 34.608406 16.080641 34.816406 16.181641 C 35.289406 16.408641 35.530031 16.914688 35.457031 17.679688 C 35.215031 20.202687 33.253938 33.008969 32.835938 34.292969 C 32.477938 35.390969 32.100813 36.335938 31.382812 36.335938 C 30.664813 36.335938 29.880422 36.08425 29.107422 35.53125 C 28.334422 34.97925 23.201281 31.536891 22.488281 31.087891 C 21.863281 30.693891 21.201813 29.711719 22.132812 28.761719 C 22.899812 27.979719 28.717844 22.332938 29.214844 21.835938 C 29.584844 21.464938 29.411828 21.017578 29.048828 21.017578 C 28.923828 21.017578 28.774141 21.070266 28.619141 21.197266 C 28.011141 21.694266 19.534781 27.366266 18.800781 27.822266 C 18.314781 28.124266 17.56225 28.341797 16.78125 28.341797 C 16.44825 28.341797 16.111109 28.301891 15.787109 28.212891 C 14.659109 27.901891 12.750187 27.322734 11.992188 27.052734 C 11.263188 26.792734 10.998047 26.543656 10.998047 26.097656 C 10.998047 25.463656 11.892938 25.026 12.835938 24.625 C 13.831938 24.202 31.066062 16.883437 32.414062 16.398438 C 33.038062 16.172438 33.608891 16.035156 34.087891 16.035156 z" strokeWidth="1" stroke="currentColor" />
  </svg>
);

export const PushoverIcon = () => (
  <svg {...commonSVGProps} viewBox="0 0 600 600">
    <path d="M 280.949 172.514 L 355.429 162.714 L 282.909 326.374 L 282.909 326.374 C 295.649 325.394 308.142 321.067 320.389 313.394 L 320.389 313.394 L 320.389 313.394 C 332.642 305.714 343.916 296.077 354.209 284.484 L 354.209 284.484 L 354.209 284.484 C 364.496 272.884 373.396 259.981 380.909 245.774 L 380.909 245.774 L 380.909 245.774 C 388.422 231.561 393.812 217.594 397.079 203.874 L 397.079 203.874 L 397.079 203.874 C 399.039 195.381 399.939 187.214 399.779 179.374 L 399.779 179.374 L 399.779 179.374 C 399.612 171.534 397.569 164.674 393.649 158.794 L 393.649 158.794 L 393.649 158.794 C 389.729 152.914 383.766 148.177 375.759 144.584 L 375.759 144.584 L 375.759 144.584 C 367.759 140.991 356.899 139.194 343.179 139.194 L 343.179 139.194 L 343.179 139.194 C 327.172 139.194 311.409 141.807 295.889 147.034 L 295.889 147.034 L 295.889 147.034 C 280.376 152.261 266.002 159.857 252.769 169.824 L 252.769 169.824 L 252.769 169.824 C 239.542 179.784 228.029 192.197 218.229 207.064 L 218.229 207.064 L 218.229 207.064 C 208.429 221.924 201.406 238.827 197.159 257.774 L 197.159 257.774 L 197.159 257.774 C 195.526 263.981 194.546 268.961 194.219 272.714 L 194.219 272.714 L 194.219 272.714 C 193.892 276.474 193.812 279.577 193.979 282.024 L 193.979 282.024 L 193.979 282.024 C 194.139 284.477 194.462 286.357 194.949 287.664 L 194.949 287.664 L 194.949 287.664 C 195.442 288.971 195.852 290.277 196.179 291.584 L 196.179 291.584 L 196.179 291.584 C 179.519 291.584 167.349 288.234 159.669 281.534 L 159.669 281.534 L 159.669 281.534 C 151.996 274.841 150.119 263.164 154.039 246.504 L 154.039 246.504 L 154.039 246.504 C 157.959 229.191 166.862 212.694 180.749 197.014 L 180.749 197.014 L 180.749 197.014 C 194.629 181.334 211.122 167.531 230.229 155.604 L 230.229 155.604 L 230.229 155.604 C 249.342 143.684 270.249 134.214 292.949 127.194 L 292.949 127.194 L 292.949 127.194 C 315.656 120.167 337.789 116.654 359.349 116.654 L 359.349 116.654 L 359.349 116.654 C 378.296 116.654 394.219 119.347 407.119 124.734 L 407.119 124.734 L 407.119 124.734 C 420.026 130.127 430.072 137.234 437.259 146.054 L 437.259 146.054 L 437.259 146.054 C 444.446 154.874 448.936 165.164 450.729 176.924 L 450.729 176.924 L 450.729 176.924 C 452.529 188.684 451.959 200.934 449.019 213.674 L 449.019 213.674 L 449.019 213.674 C 445.426 229.027 438.646 244.464 428.679 259.984 L 428.679 259.984 L 428.679 259.984 C 418.719 275.497 406.226 289.544 391.199 302.124 L 391.199 302.124 L 391.199 302.124 C 376.172 314.697 358.939 324.904 339.499 332.744 L 339.499 332.744 L 339.499 332.744 C 320.066 340.584 299.406 344.504 277.519 344.504 L 277.519 344.504 L 275.069 344.504 L 212.839 484.154 L 142.279 484.154 L 280.949 172.514 Z" />
  </svg>
);

export const GotifyIcon = () => (
  <svg {...commonSVGProps} viewBox="0 0 140 140">
    <path d="m 114.5,21.4 c -11.7,0 -47.3,5.9 -54.3,7.1 -47.3,8.0 -48.4,9.9 -50.1,12.8 -1.2,2.1 -2.4,4.0 2.6,29.4 2.3,11.5 5.8,26.9 8.8,35.8 1.8,5.4 3.6,8.8 6.9,10.1 0.8,0.3 1.7,0.5 2.7,0.6 0.2,0.0 0.3,0.0 0.5,0.0 12.8,0 89.1,-19.5 89.9,-19.7 1.4,-0.4 4.0,-1.5 5.3,-5.1 1.8,-4.7 1.9,-16.7 0.5,-35.7 -2.1,-28.0 -4.1,-31.0 -4.8,-32.0 -2.0,-3.1 -5.6,-3.3 -6.7,-3.3 -0.4,-0.0 -0.9,-0.0 -1.4,-0.0 z m -1.9,6.6 c -9.3,12.0 -18.9,24.0 -25.9,32.4 -2.3,2.8 -4.3,5.1 -6.0,7.0 -1.7,1.9 -2.9,3.2 -3.8,4.0 l -0.3,0.3 -0.4,-0.1 c -1.0,-0.3 -2.5,-0.9 -4.4,-1.7 -2.3,-1.0 -5.2,-2.3 -8.8,-3.9 C 51.6,60.7 34.4,52.2 18.0,43.6 30.3,39.7 95.0,28.7 112.6,27.9 Z m 5.7,5.0 c 2.0,11.8 4.5,42.6 3.1,54.0 -1.8,-1.4 -10.1,-8.0 -19.8,-15.2 -3.0,-2.3 -5.9,-4.3 -8.4,-6.1 l -0.7,-0.5 0.5,-0.6 C 99.5,56.9 108.0,46.2 118.3,32.9 Z M 16.1,51.1 c 3.0,1.5 14.3,7.4 27.4,13.8 5.3,2.6 9.9,4.8 13.9,6.7 l 0.9,0.4 -0.7,0.8 C 50.3,81.2 40.6,92.8 28.8,107.2 24.5,96.7 17.9,65.0 16.1,51.1 Z m 71.5,19.7 0.6,0.4 c 7.8,5.5 18.1,13.2 27.9,21.0 C 104.9,95.1 53.2,107.9 36.0,110.3 46.6,97.4 57.3,84.7 65.1,75.8 l 0.4,-0.4 0.5,0.2 c 5.7,2.5 9.3,3.7 11.1,3.8 0.1,0.0 0.2,0.0 0.3,0.0 0.6,0 1.0,-0.1 1.4,-0.3 0.6,-0.2 2.0,-0.7 8.3,-7.7 z" />
  </svg>
);

export const NtfyIcon = () => (
  <svg  {...commonSVGProps} viewBox="0 0 50.8 50.8" xmlns="http://www.w3.org/2000/svg">
    <path d="M44.98 39.952V10.848H7.407v27.814l-1.587 4.2 8.393-2.91Z" />
    <path d="M27.781 31.485h8.202" />
    <path d="m65.979 100.011 9.511 5.492-9.511 5.491" transform="translate(-51.81 -80.758)" />
  </svg>
);

export const LunaSeaIcon = () => (
  <svg {...commonSVGProps} viewBox="0 0 750 750">
    <path d="m554.69 180.46c-333.63 0-452.75 389.23-556.05 389.23 185.37 0 237.85-247.18 419.12-247.18l47.24-102.05z" />
    <path d="m749.31 375.08c0 107.48-87.14 194.61-194.62 194.61s-194.62-87.13-194.62-194.61 87.13-194.62 194.62-194.62c7.391-2e-3 14.776 0.412 22.12 1.24-78.731 10.172-136.59 78.893-133.2 158.2 3.393 79.313 66.907 142.84 146.22 146.25 79.311 3.411 148.05-54.43 158.24-133.16 0.826 7.331 1.24 14.703 1.24 22.08z" />
  </svg>
);
