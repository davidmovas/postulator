"use client"

import { AlertCircleIcon, UploadIcon, XIcon, FileTextIcon } from "lucide-react"
import * as React from "react"
import { useFileUpload } from "@/hooks/use-file-upload"
import { Button } from "@/components/ui/button"

export type FileUploadProps = {
  accept?: string; // comma-separated without dots, e.g. "txt,csv,json,xls,xlsx"
  multiple?: boolean;
  onFileSelected?: (file: File | null) => void;
}

export default function FileUpload({ accept = "txt,csv,json,xls,xlsx", multiple = false, onFileSelected }: FileUploadProps) {
  const [
    { files, isDragging, errors },
    {
      handleDragEnter,
      handleDragLeave,
      handleDragOver,
      handleDrop,
      openFileDialog,
      removeFile,
      getInputProps,
    },
  ] = useFileUpload({
    accept,
    multiple,
  })

  const previewUrl = files[0]?.preview || null

  React.useEffect(() => {
    if (!onFileSelected) return;
    const candidate = files[0]?.file as unknown;
    const f = candidate && typeof candidate === "object" && "arrayBuffer" in (candidate as object) ? (candidate as File) : null;
    onFileSelected(f);
  }, [files, onFileSelected]);

  return (
    <div className="flex flex-col gap-2">
      <div className="relative">
        {/* Зона перетаскивания */}
        <div
          onDragEnter={handleDragEnter}
          onDragLeave={handleDragLeave}
          onDragOver={handleDragOver}
          onDrop={handleDrop}
          data-dragging={isDragging || undefined}
          className="border-input data-[dragging=true]:bg-accent/50 has-[input:focus]:border-ring has-[input:focus]:ring-ring/50 relative flex min-h-52 flex-col items-center justify-center overflow-hidden rounded-xl border border-dashed p-4 transition-colors has-[input:focus]:ring-[3px]"
        >
          <input
            {...getInputProps()}
            className="sr-only"
            aria-label="Upload file"
          />
          {previewUrl ? (
            <div className="absolute inset-0 flex items-center justify-center p-4">
              {/* Не все файлы имеют превью; если нет - показываем иконку */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={previewUrl}
                alt={files[0]?.file?.name || "Uploaded file"}
                className="mx-auto max-h-full rounded object-contain"
                onError={(e) => {
                  // Скрыть битое изображение превью, если файл не изображение
                  (e.currentTarget as HTMLImageElement).style.display = "none";
                }}
              />
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center px-4 py-3 text-center">
              <div
                className="bg-background mb-2 flex size-11 shrink-0 items-center justify-center rounded-full border"
                aria-hidden="true"
              >
                <FileTextIcon className="size-4 opacity-60" />
              </div>
              <p className="mb-1.5 text-sm font-medium">Drop your file here</p>
              <p className="text-muted-foreground text-xs">
                Supported formats: TXT, CSV, JSON, XLS/XLSX
              </p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={openFileDialog}
              >
                <UploadIcon
                  className="-ms-1 size-4 opacity-60"
                  aria-hidden="true"
                />
                Choose file
              </Button>
            </div>
          )}
        </div>

        {files[0] && (
          <div className="absolute top-4 right-4">
            <button
              type="button"
              className="focus-visible:border-ring focus-visible:ring-ring/50 z-50 flex size-8 cursor-pointer items-center justify-center rounded-full bg-black/60 text-white transition-[color,box-shadow] outline-none hover:bg-black/80 focus-visible:ring-[3px]"
              onClick={() => removeFile(files[0]?.id)}
              aria-label="Remove file"
            >
              <XIcon className="size-4" aria-hidden="true" />
            </button>
          </div>
        )}
      </div>

      {errors.length > 0 && (
        <div
          className="text-destructive flex items-center gap-1 text-xs"
          role="alert"
        >
          <AlertCircleIcon className="size-3 shrink-0" />
          <span>{errors[0]}</span>
        </div>
      )}
    </div>
  )
}
