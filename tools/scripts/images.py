import requests
from PIL import Image
import io
import os

image_urls = [
    
]

file_names = [
    # "cat1_thumbnail.png", "cat2_thumbnail.png", "cat3_thumbnail.png", "cat4_thumbnail.png",
    # "cat5_thumbnail.png", "cat6_thumbnail.png", "cat7_thumbnail.png", "cat8_thumbnail.png",
    # "dog1_thumbnail.png", "dog2_thumbnail.png", "dog3_thumbnail.png", "dog4_thumbnail.png",
    # "dog5_thumbnail.png", "dog6_thumbnail.png", "dog7_thumbnail.png", "dog8_thumbnail.png",
    # "shop1_thumbnail.png", "shop2_thumbnail.png", "shop3_thumbnail.png", "shop4_thumbnail.png",
    # "shop5_thumbnail.png", "shop6_thumbnail.png", "shop7_thumbnail.png", "shop8_thumbnail.png",
    # "bike1_thumbnail.png", "bike2_thumbnail.png", "bike3_thumbnail.png", "bike4_thumbnail.png",
    # "bike5_thumbnail.png", "bike6_thumbnail.png", "bike7_thumbnail.png", "bike8_thumbnail.png",
    # "flower1_thumbnail.png", "flower2_thumbnail.png", "flower3_thumbnail.png", "flower4_thumbnail.png",
    # "flower5_thumbnail.png", "flower6_thumbnail.png", "flower7_thumbnail.png", "flower8_thumbnail.png"
    "vegetable1_thumbnail.png", "vegetable2_thumbnail.png", "vegetable3_thumbnail.png", "vegetable4_thumbnail.png",
    "vegetable5_thumbnail.png", "vegetable6_thumbnail.png", "vegetable7_thumbnail.png", "vegetable8_thumbnail.png"

]

for i, (url, filename) in enumerate(zip(image_urls, file_names)):
    try:
        print(f"Processing {i+1}/16: {filename}")
        response = requests.get(url)
        response.raise_for_status()
        
        image = Image.open(io.BytesIO(response.content))
        
        # image.thumbnail((200, 200), Image.Resampling.LANCZOS)
        
        width, height = image.size
        crop_size = min(width, height)
        left = (width - crop_size) // 2
        top = (height - crop_size) // 2
        right = left+crop_size
        bottom = top + crop_size

        cropped = image.crop((left, top, right, bottom))
        thumbnail = cropped.resize((200, 200), Image.Resampling.LANCZOS)
        thumbnail.save(filename, "JPEG", quality=90, optimize=True)
        print(f"✓ {filename}")
        
    except Exception as e:
        print(f"✗ Error of processing {filename}: {str(e)}")

print("\nDone!")