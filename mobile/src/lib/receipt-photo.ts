import type { ImagePickerAsset, ImagePickerOptions } from 'expo-image-picker';

import type { ReceiptPhoto } from '@/lib/receipts-api';

export async function pickReceiptPhoto(source: 'camera' | 'library'): Promise<ReceiptPhoto | null> {
  let picker: typeof import('expo-image-picker');
  try {
    picker = await import('expo-image-picker');
  } catch {
    throw new Error('The photo picker is missing from this app build. Rebuild and install the development APK, then try again.');
  }

  if (source === 'camera') {
    const permission = await picker.requestCameraPermissionsAsync();
    if (!permission.granted) {
      throw new Error(permission.canAskAgain
        ? 'Allow camera access to photograph a receipt, or choose an existing photo.'
        : 'Camera access is disabled. Enable it in your phone’s app settings, or choose an existing photo.');
    }
  }

  const options: ImagePickerOptions = { mediaTypes: ['images'], quality: 0.8, allowsEditing: false, exif: false, base64: false };
  const result = source === 'camera'
    ? await picker.launchCameraAsync(options)
    : await picker.launchImageLibraryAsync(options);
  if (result.canceled) return null;

  return receiptPhoto(result.assets[0]);
}

function receiptPhoto(asset: ImagePickerAsset): ReceiptPhoto {
  if (asset.fileSize !== undefined && asset.fileSize > 10 * 1024 * 1024) {
    throw new Error('This photo exceeds 10 MiB. Use a smaller image or take a photo in standard-resolution mode.');
  }

  if (asset.width * asset.height > 25_000_000) {
    throw new Error('This photo exceeds 25 megapixels. Take another using your camera’s standard-resolution mode.');
  }

  const path = asset.fileName ?? asset.uri.split('?')[0];
  const detected = asset.mimeType ?? (/\.png$/i.test(path) ? 'image/png' : /\.jpe?g$/i.test(path) ? 'image/jpeg' : null);
  const mimeType = detected === 'image/jpg' ? 'image/jpeg' : detected;
  if (mimeType !== 'image/jpeg' && mimeType !== 'image/png') {
    throw new Error('Choose a JPEG or PNG receipt photo. Other image formats are not supported yet.');
  }

  return { uri: asset.uri, mimeType, name: mimeType === 'image/png' ? 'receipt.png' : 'receipt.jpg' };
}
