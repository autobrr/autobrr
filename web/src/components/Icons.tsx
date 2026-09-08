/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */


interface IconProps {
    className?: string;
}

export const OpenIdIcon = ({ className }: IconProps) => (
  <svg className={className} fill="currentColor" viewBox="0 0 448 512" aria-hidden="true" xmlns="http://www.w3.org/2000/svg"><path d="M271.5 432l-68 32C88.5 453.7 0 392.5 0 318.2 0 246.7 82.5 187.2 191.7 173.9l0 43c-71.5 12.5-124 53-124 101.3 0 51 58.5 93.3 135.7 103l0-340 68-33.2 0 384 .1 0zM448 291l-131.3-28.5 36.8-20.7c-19.5-11.5-43.5-20-70-24.8l0-43c46.2 5.5 87.7 19.5 120.3 39.3l35-19.8 9.2 97.5z"></path></svg>
);

export const SortIcon = ({ className }: IconProps) => (
  <svg className={className} stroke="currentColor" fill="currentColor" strokeWidth="0" viewBox="0 0 320 512" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M41 288h238c21.4 0 32.1 25.9 17 41L177 448c-9.4 9.4-24.6 9.4-33.9 0L24 329c-15.1-15.1-4.4-41 17-41zm255-105L177 64c-9.4-9.4-24.6-9.4-33.9 0L24 183c-15.1 15.1-4.4 41 17 41h238c21.4 0 32.1-25.9 17-41z"></path></svg>
);

export const SortUpIcon = ({ className }: IconProps) => (
  <svg className={className} stroke="currentColor" fill="currentColor" strokeWidth="0" viewBox="0 0 320 512" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M279 224H41c-21.4 0-32.1-25.9-17-41L143 64c9.4-9.4 24.6-9.4 33.9 0l119 119c15.2 15.1 4.5 41-16.9 41z"></path></svg>
);

export const SortDownIcon = ({ className }: IconProps) => (
  <svg className={className} stroke="currentColor" fill="currentColor" strokeWidth="0" viewBox="0 0 320 512" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M41 288h238c21.4 0 32.1 25.9 17 41L177 448c-9.4 9.4-24.6 9.4-33.9 0L24 329c-15.1-15.1-4.4-41 17-41z"></path></svg>
);

export const RingResizeSpinner = ({ className }: IconProps) => (
  <svg className={className} stroke="currentColor" fill="currentColor" strokeWidth="3" strokeLinecap="round" viewBox="0 0 24 24" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><g><circle cx="12" cy="12" r="9.5" fill="none"><animate attributeName="stroke-dasharray" dur="1.5s" calcMode="spline" values="0 150;42 150;42 150;42 150" keyTimes="0;0.475;0.95;1" keySplines="0.42,0,0.58,1;0.42,0,0.58,1;0.42,0,0.58,1" repeatCount="indefinite"/><animate attributeName="stroke-dashoffset" dur="1.5s" calcMode="spline" values="0;-16;-59;-59" keyTimes="0;0.475;0.95;1" keySplines="0.42,0,0.58,1;0.42,0,0.58,1;0.42,0,0.58,1" repeatCount="indefinite"/></circle><animateTransform attributeName="transform" type="rotate" dur="2s" values="0 12 12;360 12 12" repeatCount="indefinite"/></g></svg>
);
