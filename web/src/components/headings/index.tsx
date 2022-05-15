import { FC } from "react";

interface Props {
    title: string;
    subtitle: string;
}

export const TitleSubtitle: FC<Props> = ({ title, subtitle }) => (
    <div>
        <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-gray-100">{title}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{subtitle}</p>
    </div>
);