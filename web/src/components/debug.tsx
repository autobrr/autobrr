const DEBUG = ({ values }: any) => {
    if (process.env.NODE_ENV !== "development") {
        return null;
    }

    return (
        <div className="w-1/2 mx-auto mt-2 flex flex-col mt-12 mb-12">
            <pre className="mt-2 dark:text-gray-500">{JSON.stringify(values, 0 as any, 2)}</pre>
        </div>
    );
};

export default DEBUG;
