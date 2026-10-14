import React, {useState} from 'react';
import Modal from "./Modal";
import Button from "./Button";

function ConfirmDialog({title, content, isOpen, close, onSuccess, confirmLabel = 'Confirm', cancelLabel = 'Cancel'}) {

    const [isLoading, setIsLoading] = useState(false);

    const confirm = () => {
        setIsLoading(true);
        onSuccess()
            .finally(() => {
                close();
                setIsLoading(false);
            })
    }

    return (
        <Modal
            title={title}
            content={content}
            actions={
                <>
                    <Button size="sm" type="danger" className="mr-2" onClick={close}>{cancelLabel}</Button>
                    <Button size="sm" isLoading={isLoading} type="success" onClick={confirm}>{confirmLabel}</Button>
                </>
            }
            isOpen={isOpen}
        />
    );
}

export default ConfirmDialog;