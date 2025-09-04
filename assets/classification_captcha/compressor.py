import requests
from PIL import Image
import io
import os

# Названия файлов
file_names = [
    "cat1_thumbnail.png",
    "cat2_thumbnail.png", "cat3_thumbnail.png", "cat4_thumbnail.png",
    "cat5_thumbnail.png", "cat6_thumbnail.png", "cat7_thumbnail.png", "cat8_thumbnail.png",
    "dog1_thumbnail.png", "dog2_thumbnail.png", "dog3_thumbnail.png", "dog4_thumbnail.png",
    "dog5_thumbnail.png", "dog6_thumbnail.png", "dog7_thumbnail.png", "dog8_thumbnail.png",
    "bike1_thumbnail.png", "bike2_thumbnail.png", "bike3_thumbnail.png", "bike4_thumbnail.png",
    "bike5_thumbnail.png", "bike6_thumbnail.png", "bike7_thumbnail.png", "bike8_thumbnail.png",
    "flower1_thumbnail.png", "flower2_thumbnail.png", "flower3_thumbnail.png", "flower4_thumbnail.png",
    "flower5_thumbnail.png", "flower6_thumbnail.png", "flower7_thumbnail.png", "flower8_thumbnail.png",
    "vegetable1_thumbnail.png", "vegetable2_thumbnail.png", "vegetable3_thumbnail.png", "vegetable4_thumbnail.png",
    "vegetable5_thumbnail.png", "vegetable6_thumbnail.png", "vegetable7_thumbnail.png", "vegetable8_thumbnail.png"
]

print("Начинаю загрузку и обработку изображений...")

for file_name in file_names:
        src="t/"+ file_name
        dst=file_name.replace(".png",".jpg")
        print(src+" => "+dst)
        image = Image.open(src)
        image.save(dst, optimize=True,quality=50, format="JPEG")

print("\nОбработка завершена!")
