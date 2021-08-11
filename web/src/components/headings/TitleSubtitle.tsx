import React from "react";

interface Props {
    title: string;
    subtitle: string;
}

const TitleSubtitle: React.FC<Props> = ({ title, subtitle }) => (
    <div>
        <h2 className="text-lg leading-6 font-medium text-gray-900">{title}</h2>
        <p className="mt-1 text-sm text-gray-500">{subtitle}</p>
    </div>
)

export default TitleSubtitle;