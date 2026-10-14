import React, {useRef, useState} from "react";
import Button from "../../../components/Button";
import ProgressBar from "../../../components/ProgressBar";
import saves from "../../../../api/resources/saves";

const ACCEPT = ".zip";

const mergeFiles = (current, added) => {
    const byName = new Map(current.map(file => [file.name, file]));
    added.forEach(file => byName.set(file.name, file));
    return Array.from(byName.values());
};

const UploadSaveForm = ({onSuccess}) => {

    const input = useRef(null);
    const [files, setFiles] = useState([]);
    const [isDragging, setIsDragging] = useState(false);
    const [isUploading, setIsUploading] = useState(false);
    const [progress, setProgress] = useState(null);

    const addFiles = list => {
        const added = Array.from(list || []);
        if (added.length) {
            setFiles(current => mergeFiles(current, added));
        }
    };

    const clearFiles = () => {
        setFiles([]);
        if (input.current) {
            input.current.value = '';
        }
    };

    const onDrop = event => {
        event.preventDefault();
        setIsDragging(false);
        if (!isUploading) {
            addFiles(event.dataTransfer.files);
        }
    };

    const upload = () => {
        setIsUploading(true);
        setProgress({overall: 0, current: {name: files[0] ? files[0].name : '', index: 0, count: files.length, progress: 0}});

        saves.upload(files, setProgress)
            .then(onSuccess)
            .finally(() => {
                setIsUploading(false);
                setProgress(null);
                clearFiles();
            });
    };

    return (
        <form onSubmit={event => { event.preventDefault(); upload(); }}>
            <div
                onClick={() => !isUploading && input.current && input.current.click()}
                onDragOver={event => {
                    event.preventDefault();
                    if (!isUploading) {
                        setIsDragging(true);
                    }
                }}
                onDragLeave={() => setIsDragging(false)}
                onDrop={onDrop}
                className={
                    'text-center px-4 py-8 mb-4 border-2 border-dashed rounded-sm '
                    + (isUploading ? 'cursor-not-allowed opacity-50 ' : 'cursor-pointer ')
                    + (isDragging ? 'border-orange bg-gray-dark' : 'border-gray-light')
                }
            >
                <div className="text-dirty-white font-bold">Drop saves here</div>
                <div className="text-gray-light text-sm">or click to choose. Several files at once are fine.</div>
                <input
                    ref={input}
                    className="hidden"
                    type="file"
                    multiple
                    accept={ACCEPT}
                    onChange={event => addFiles(event.currentTarget.files)}
                />
            </div>

            {files.length > 0 &&
                <div className="mb-4">
                    <div className="flex justify-between text-sm mb-1">
                        <span className="text-dirty-white">
                            {files.length} {files.length === 1 ? 'file' : 'files'} ready to upload
                        </span>
                        {!isUploading &&
                            <span
                                className="text-gray-light cursor-pointer hover:text-orange"
                                onClick={clearFiles}
                            >
                                Clear
                            </span>
                        }
                    </div>
                    <ul className="text-sm text-gray-light">
                        {files.map(file => <li key={file.name}>{file.name}</li>)}
                    </ul>
                </div>
            }

            {progress &&
                <div className="mb-4">
                    <ProgressBar
                        value={progress.overall}
                        label={`Overall (${progress.current.index + 1} of ${progress.current.count})`}
                    />
                    <ProgressBar
                        className="mt-3"
                        value={progress.current.progress}
                        label={progress.current.name}
                    />
                </div>
            }

            <Button
                type="success"
                isLoading={isUploading}
                isDisabled={files.length === 0}
                onClick={upload}
            >
                Upload
            </Button>
        </form>
    )
};

export default UploadSaveForm;
