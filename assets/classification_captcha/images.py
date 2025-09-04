import requests
from PIL import Image
import io
import os

# URLs изображений котов и собак
image_urls = [
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/50916fed-e730-4588-95eb-041c0e79538e.png",  # cat1
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/50a7b799-66b8-474e-9125-a86b15234f80.png",  # cat2
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/0868b8fa-158e-4151-ad61-1de83506fd46.png",  # cat3
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/e36b3005-403b-4130-bf36-9dff1ded06c2.png",  # cat4
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/7f942193-e587-4edd-ac4e-4e84c3081f98.png",  # cat5
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/4dd436ec-fb5e-4902-82b3-3ca3c332a514.png",  # cat6
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/71fc3bb2-1a92-402e-b7b5-aac1ee581b19.png",  # cat7
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/9681c49f-d847-466f-b600-19b6cc165ca2.png",  # cat8
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/7342bd7e-5bd9-4eff-8782-ca9a944a488f.png",  # dog1
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/d76fd755-0f03-409b-bc48-6d6d18c60349.png",  # dog2
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/44693433-43bc-456a-9ed2-02c5e66f33ce.png",  # dog3
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/0868778a-e0e2-426d-afea-11bbbe43641b.png",  # dog4
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/302b9d35-ed0f-4a65-96aa-be7c0d4a569c.png",  # dog5
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/a10aaede-68f0-4ef1-8244-0e38f7bb3970.png",  # dog6
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/097a8f6d-40ab-4943-af0b-6c9a0e6c37b8.png",  # dog7
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/56702e71-3f0e-4c88-932d-280d22aab17f.png",  # dog8
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/d675cacc-b431-433d-ab2c-165db3f911be.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/a7614b83-fd0e-45d3-81d3-176456081060.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/a6461fbc-1ee2-417a-a724-2f4e644b533a.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/978f06af-f527-47b2-a2b9-c6cf4854949d.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/ac4a5fca-0115-4500-9882-b54584f92744.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/5527f34d-9d36-4bee-ad3f-5b58c3c84887.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/646497e1-7e40-4f8f-8c78-a6f26d8ca285.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/ff36dd9d-de71-4f7d-ac36-8bc8afdfe7be.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/f1dca1e4-b891-4370-8c36-8faeccfc8cc6.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/9f58b62c-13b3-4a1f-bc9b-68cdbb9a0655.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/3937d93a-0691-44f2-9ecf-ab269b60043f.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/c7ef418b-842e-45ed-8fb8-dfd20e1f4b86.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/cd60b92f-2c80-4c13-beb3-1c26d53156d9.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/98692e3f-011c-45dd-b28b-341615de081b.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/f7e63b5b-d732-4d3f-a5f7-ce7f8e07ecad.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/e75d13ca-6fa6-4584-be4a-d50cf062c8da.png",

    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/65600638-a463-42e9-a63b-a2461292ceb3.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/697ed2af-f10d-461f-b5c7-ea9c3a230440.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/acd27e53-e485-43ef-83a0-b3f9d6a9ab04.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/b7466c67-6f0b-4ef3-8f0c-5f7738132ca6.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/328e2915-495f-4b3b-abb3-62cb177d4c41.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/3b01c58c-f7a5-4408-96a3-c584b644708d.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/a5c8148e-4b19-4ef5-b2fb-8be7dc1f5a45.png",
    # "https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/5e593270-c531-4f87-a409-0b8746745b51.png",

"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/2b1bfd1e-7edd-4ca6-83b1-95e36c84f5f9.png",
"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/6c032ff2-9eb4-4d88-921f-75807db61980.png",
"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/432d147e-7c66-407f-8c9d-1de8f52bfc8d.png",
"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/040c5a5e-a3be-4f93-8a7a-749b25fb5dac.png",
"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/4eb85bcc-7187-4018-9d9c-cfe57eb48587.png",
"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/740f48ba-64e5-4b33-bf14-41aeeee05490.png",
"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/2acfe2cf-e3fb-4e68-a73f-878562589c9c.png",
"https://user-gen-media-assets.s3.amazonaws.com/gpt4o_images/ae022ea6-7682-4ea1-a951-9d1ea3c5f969.png",
]

# Названия файлов
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

print("Начинаю загрузку и обработку изображений...")

# Создаем превью для каждого изображения
for i, (url, filename) in enumerate(zip(image_urls, file_names)):
    try:
        # Скачиваем изображение
        print(f"Обрабатываю изображение {i+1}/16: {filename}")
        response = requests.get(url)
        response.raise_for_status()
        
        # Открываем изображение
        image = Image.open(io.BytesIO(response.content))
        
        # Создаем превью 200x200 пикселей с сохранением пропорций
        # image.thumbnail((200, 200), Image.Resampling.LANCZOS)
        
        # Создаем квадратное изображение 200x200 с центрированием
        width, height = image.size
        crop_size = min(width, height)
        left = (width - crop_size) // 2
        top = (height - crop_size) // 2
        right = left+crop_size
        bottom = top + crop_size

        cropped = image.crop((left, top, right, bottom))
        thumbnail = cropped.resize((200, 200), Image.Resampling.LANCZOS)

        # Сохраняем превью
        thumbnail.save(filename, "JPEG", quality=90, optimize=True)
        print(f"✓ Сохранено: {filename} (200x200)")
        
    except Exception as e:
        print(f"✗ Ошибка при обработке {filename}: {str(e)}")

print("\nОбработка завершена!")
print(f"Создано превью изображений размером 200x200 пикселей")