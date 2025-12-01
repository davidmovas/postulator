import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    UploadMedia,
    UploadMediaFromURL,
    GetMedia,
    DeleteMedia,
} from "@/wailsjs/wailsjs/go/handlers/MediaHandler";
import { unwrapResponse } from "@/lib/api-utils";

export interface MediaResult {
    id: number;
    sourceUrl: string;
    altText: string;
}

function mapMediaResult(x: dto.MediaResult): MediaResult {
    return {
        id: x.id,
        sourceUrl: x.sourceUrl,
        altText: x.altText,
    };
}

export const mediaService = {
    /**
     * Upload a file to WordPress media library
     * @param siteId - The site ID
     * @param filename - The filename with extension
     * @param fileData - Base64 encoded file content
     * @param altText - Alt text for the image
     */
    async uploadMedia(siteId: number, filename: string, fileData: string, altText: string = ""): Promise<MediaResult> {
        const response = await UploadMedia(siteId, filename, fileData, altText);
        const result = unwrapResponse<dto.MediaResult>(response);
        return mapMediaResult(result);
    },

    /**
     * Download an image from URL and upload to WordPress
     * @param siteId - The site ID
     * @param imageUrl - The URL of the image to download and upload
     * @param altText - Alt text for the image
     */
    async uploadMediaFromURL(siteId: number, imageUrl: string, altText: string = ""): Promise<MediaResult> {
        const response = await UploadMediaFromURL(siteId, imageUrl, altText);
        const result = unwrapResponse<dto.MediaResult>(response);
        return mapMediaResult(result);
    },

    /**
     * Get media information from WordPress
     * @param siteId - The site ID
     * @param mediaId - The WordPress media ID
     */
    async getMedia(siteId: number, mediaId: number): Promise<MediaResult> {
        const response = await GetMedia(siteId, mediaId);
        const result = unwrapResponse<dto.MediaResult>(response);
        return mapMediaResult(result);
    },

    /**
     * Delete media from WordPress
     * @param siteId - The site ID
     * @param mediaId - The WordPress media ID
     */
    async deleteMedia(siteId: number, mediaId: number): Promise<void> {
        const response = await DeleteMedia(siteId, mediaId);
        unwrapResponse<string>(response);
    },

    /**
     * Convert a File object to base64 string
     */
    fileToBase64(file: File): Promise<string> {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => {
                const result = reader.result as string;
                // Remove the data URL prefix (e.g., "data:image/png;base64,")
                const base64 = result.split(',')[1];
                resolve(base64);
            };
            reader.onerror = reject;
            reader.readAsDataURL(file);
        });
    },
};
