// Современный фронтенд для садоводческого блога с REST API

// Состояние приложения
const AppState = {
    currentPage: 'home',
    currentArticleId: null,
    articles: [],
    articlesLoaded: false,
    currentArticle: null,
    loading: false,
    error: null
};

// API базовый URL (в реальном проекте это будет URL вашего бэкенда)
const API_BASE_URL = '';

// Заглушка данных для случая недоступности API
const FALLBACK_DATA = {
    articles: [
        {
            id: 1,
            title: "7 главных советов начинающим садоводам",
            author: "Анна Петрова", 
            date: "2024-08-15",
            content: "Друзья, хочу поделиться основными правилами для тех, кто только начинает свой путь в садоводстве. Первое - не торопитесь с высадкой, соблюдайте температурный режим. Второе - замачивайте семена только для рассады, а в открытый грунт сейте сухие семена. Третье - следите за влажностью почвы, но не переувлажняйте. Четвертое - изучите информацию о растениях заранее. И помните - не бросайте сад после уборки урожая!",
            likes: 42,
            commentsCount: 8
        },
        {
            id: 2,
            title: "Неприхотливые овощи для начинающих огородников",
            author: "Игорь Смирнов",
            date: "2024-08-14", 
            content: "Если вы новичок в огородничестве, начните с простых культур. Кабачки - даже один куст обеспечит семью урожаем с июля до морозов. Нужен только полив раз в неделю. Тыква настолько неприхотлива, что растет даже в траве. Свекла, репчатый лук, редис - все это можно сеять прямо в грунт. Не забывайте про зелень: салат, укроп, петрушка - растут быстро и почти без ухода.",
            likes: 38,
            commentsCount: 6
        },
        {
            id: 3,
            title: "Автоматический полив - спасение для занятых садоводов",
            author: "Елена Козлова",
            date: "2024-08-13",
            content: "Хочу поделиться опытом установки автополива. После установки системы капельного полива жизнь стала намного проще! Растения получают воду регулярно, даже когда меня нет дома. Система окупилась за один сезон - экономия воды и времени огромная. Особенно рекомендую для теплиц и контейнерных растений. Можно даже самим сделать простую систему из пластиковых бутылок.",
            likes: 29,
            commentsCount: 5
        }
    ],
    comments: {
        1: [
            {id: 1, author: "Михаил С.", date: "2024-08-15", content: "Отличные советы! Особенно про температурный режим - сам когда-то поторопился и потерял почти всю рассаду.", likes: 5},
            {id: 2, author: "Елена В.", date: "2024-08-15", content: "А я всегда замачиваю все семена, теперь понимаю свои ошибки. Спасибо за информацию!", likes: 3}
        ],
        2: [
            {id: 9, author: "Галина П.", date: "2024-08-14", content: "Кабачки действительно супер! У меня три куста, и я уже устала их собирать)", likes: 7}
        ],
        3: [
            {id: 15, author: "Петр Н.", date: "2024-08-13", content: "А сколько примерно стоит установка такой системы для теплицы 3х6?", likes: 2}
        ]
    }
};

// Утилиты
const Utils = {
    formatDate(dateString) {
        const date = new Date(dateString);
        const options = { year: 'numeric', month: 'long', day: 'numeric' };
        return date.toLocaleDateString('ru-RU', options);
    },

    truncateText(text, maxLength = 200) {
        if (text.length <= maxLength) return text;
        return text.slice(0, maxLength) + '...';
    },

    showElement(elementId) {
        const element = document.getElementById(elementId);
        if (element) element.style.display = 'block';
    },

    hideElement(elementId) {
        const element = document.getElementById(elementId);
        if (element) element.style.display = 'none';
    },

    setLoading(pageId, isLoading) {
        const loadingId = `${pageId}-loading`;
        const errorId = `${pageId}-error`;
        
        if (isLoading) {
            this.showElement(loadingId);
            this.hideElement(errorId);
        } else {
            this.hideElement(loadingId);
        }
    }
};

// API функции
const API = {
    async request(endpoint, options = {}) {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 10000);

        try {
            const response = await fetch(`${API_BASE_URL}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                signal: controller.signal,
                ...options
            });

            clearTimeout(timeoutId);

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            clearTimeout(timeoutId);
            console.warn(`API request failed: ${error.message}. Using fallback data.`);
            throw error;
        }
    },

    async getArticles() {
        try {
            const response = await this.request('/articles');
            return response.articles || response;
        } catch (error) {
            return FALLBACK_DATA.articles;
        }
    },

    async getArticle(id) {
        try {
            const response = await this.request(`/articles/${id}`);
            return response;
        } catch (error) {
            const article = FALLBACK_DATA.articles.find(a => a.id === parseInt(id));
            if (article) {
                return {
                    ...article,
                    comments: FALLBACK_DATA.comments[id] || []
                };
            }
            throw error;
        }
    },

    async likeArticle(id) {
        try {
            await this.request(`/articles/${id}/like`, { method: 'POST' });
            return true;
        } catch (error) {
            console.warn('Like API unavailable, simulating success');
            return true;
        }
    },

    async addComment(articleId, comment) {
        try {
            const response = await this.request(`/articles/${articleId}/comments`, {
                method: 'POST',
                body: JSON.stringify(comment)
            });
            return response;
        } catch (error) {
            console.warn('Comment API unavailable, simulating success');
            const newComment = {
                id: Date.now(),
                author: comment.author,
                content: comment.content,
                date: new Date().toISOString().split('T')[0],
                likes: 0
            };
            return newComment;
        }
    },

    async likeComment(articleId, commentId) {
        try {
            await this.request(`/articles/${articleId}/comments/${commentId}/like`, { method: 'POST' });
            return true;
        } catch (error) {
            console.warn('Comment like API unavailable, simulating success');
            return true;
        }
    }
};

// Обновление статьи в состоянии приложения
function updateArticleInState(articleId, updates) {
    // Обновляем текущую статью если она открыта
    if (AppState.currentArticle && AppState.currentArticle.id === articleId) {
        Object.assign(AppState.currentArticle, updates);
    }

    // Обновляем статью в общем списке
    const articleInList = AppState.articles.find(a => a.id === articleId);
    if (articleInList) {
        Object.assign(articleInList, updates);
    }

    // Обновляем счетчики на карточках если главная страница отображается
    updateCardCounters(articleId);
}

// Обновление счетчиков на карточках
function updateCardCounters(articleId) {
    const likesEl = document.getElementById(`card-likes-${articleId}`);
    const commentsEl = document.getElementById(`card-comments-${articleId}`);
    
    const article = AppState.articles.find(a => a.id === articleId);
    if (!article) return;

    if (likesEl) {
        likesEl.textContent = article.likes;
    }
    if (commentsEl) {
        commentsEl.textContent = article.commentsCount;
    }
}

// Навигационные функции
window.navigateToHome = function() {
    console.log('Navigating to home');
    AppState.currentPage = 'home';
    AppState.currentArticleId = null;
    
    Utils.hideElement('article-page');
    Utils.showElement('home-page');
    
    // При переходе на главную всегда перерисовываем карточки с актуальными данными
    if (AppState.articlesLoaded) {
        renderArticleCards(AppState.articles);
    } else {
        loadArticles();
    }
};

window.navigateToArticle = function(articleId) {
    console.log('Navigating to article:', articleId);
    AppState.currentPage = 'article';
    AppState.currentArticleId = parseInt(articleId);
    
    Utils.hideElement('home-page');
    Utils.showElement('article-page');
    
    loadArticle(parseInt(articleId));
};

// Обработчики лайков с независимой логикой
window.handleArticleLike = async function(articleId, btn) {
    console.log('Article like for:', articleId);
    
    if (btn.dataset.busy === 'true') return;
    
    btn.dataset.busy = 'true';
    btn.disabled = true;
    
    try {
        const likeCount = btn.querySelector('.like-count');
        const currentLikes = parseInt(likeCount.textContent);
        const newLikes = currentLikes + 1;
        
        btn.classList.add('liked');
        likeCount.textContent = newLikes;

        await API.likeArticle(articleId);
        updateArticleInState(parseInt(articleId), { likes: newLikes });
        
    } catch (error) {
        // Откатываем изменения при ошибке
        const likeCount = btn.querySelector('.like-count');
        const currentLikes = parseInt(likeCount.textContent);
        btn.classList.remove('liked');
        likeCount.textContent = currentLikes - 1;
        console.error('Ошибка при отправке лайка:', error);
    } finally {
        btn.dataset.busy = 'false';
        btn.disabled = false;
    }
};

window.handleCardLike = async function(articleId, btn) {
    console.log('Card like for article:', articleId);
    
    if (btn.dataset.busy === 'true') return;
    
    btn.dataset.busy = 'true';
    btn.disabled = true;
    
    try {
        btn.classList.add('liked');
        btn.textContent = '✅ Нравится';
        
        const card = btn.closest('.article-card');
        const likesCountEl = card.querySelector('.article-likes-count');
        const currentLikes = parseInt(likesCountEl.textContent);
        const newLikes = currentLikes + 1;
        likesCountEl.textContent = newLikes;

        await API.likeArticle(articleId);
        updateArticleInState(articleId, { likes: newLikes });
        
    } catch (error) {
        // Откатываем изменения при ошибке
        btn.classList.remove('liked');
        btn.textContent = '👍 Нравится';
        const card = btn.closest('.article-card');
        const likesCountEl = card.querySelector('.article-likes-count');
        const currentLikes = parseInt(likesCountEl.textContent);
        likesCountEl.textContent = currentLikes - 1;
        console.error('Ошибка при отправке лайка:', error);
    } finally {
        btn.dataset.busy = 'false';
        btn.disabled = false;
    }
};

window.handleCommentLike = async function(articleId, commentId, btn) {
    console.log('Comment like:', articleId, commentId);
    
    if (btn.dataset.busy === 'true') return;
    
    btn.dataset.busy = 'true';
    btn.disabled = true;
    
    try {
        const likeCount = btn.querySelector('.comment-like-count');
        const currentLikes = parseInt(likeCount.textContent);
        const newLikes = currentLikes + 1;
        
        btn.classList.add('liked');
        likeCount.textContent = newLikes;

        await API.likeComment(articleId, commentId);
        
        // Обновляем лайки комментария в состоянии
        if (AppState.currentArticle && AppState.currentArticle.comments) {
            const comment = AppState.currentArticle.comments.find(c => c.id === commentId);
            if (comment) {
                comment.likes = newLikes;
            }
        }
        
    } catch (error) {
        // Откатываем изменения при ошибке
        btn.classList.remove('liked');
        const likeCount = btn.querySelector('.comment-like-count');
        const currentLikes = parseInt(likeCount.textContent);
        likeCount.textContent = currentLikes - 1;
        console.error('Ошибка при лайке комментария:', error);
    } finally {
        btn.dataset.busy = 'false';
        btn.disabled = false;
    }
};

window.handleAddComment = async function(event) {
    event.preventDefault();
    console.log('Adding comment');
    
    const form = event.target;
    const authorInput = form.querySelector('#comment-author');
    const contentInput = form.querySelector('#comment-content');
    
    const author = authorInput.value.trim();
    const content = contentInput.value.trim();
    
    if (!author || !content) {
        alert('Пожалуйста, заполните все поля');
        return;
    }

    const submitBtn = form.querySelector('button[type="submit"]');
    const originalText = submitBtn.textContent;
    submitBtn.disabled = true;
    submitBtn.textContent = 'Добавление...';

    try {
        const newComment = await API.addComment(AppState.currentArticleId, {
            author,
            content
        });

        if (!AppState.currentArticle.comments) {
            AppState.currentArticle.comments = [];
        }
        AppState.currentArticle.comments.push(newComment);

        const newCommentsCount = AppState.currentArticle.comments.length;
        updateArticleInState(AppState.currentArticleId, { commentsCount: newCommentsCount });

        renderComments(AppState.currentArticle.comments);
        
        const commentsCount = document.querySelector('.comments-count');
        if (commentsCount) {
            commentsCount.textContent = newCommentsCount;
        }

        form.reset();

        setTimeout(() => {
            const commentsList = document.getElementById('comments-list');
            const lastComment = commentsList.lastElementChild;
            if (lastComment && lastComment.classList.contains('comment')) {
                lastComment.scrollIntoView({ behavior: 'smooth', block: 'center' });
                lastComment.style.background = 'var(--color-bg-3)';
                setTimeout(() => {
                    lastComment.style.background = '';
                }, 2000);
            }
        }, 100);

    } catch (error) {
        alert('Ошибка при добавлении комментария. Попробуйте еще раз.');
        console.error('Ошибка добавления комментария:', error);
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = originalText;
    }
};

// Рендеринг компонентов
function renderArticleCard(article) {
    return `
        <div class="article-card" data-article-id="${article.id}">
            <header class="article-card-header">
                <h2 class="article-card-title" onclick="navigateToArticle(${article.id})" style="cursor: pointer;">${article.title}</h2>
                <div class="article-card-meta">
                    <span class="article-card-author">${article.author}</span>
                    <span class="article-card-date">${Utils.formatDate(article.date)}</span>
                </div>
                <div class="article-card-stats">
                    <span class="stat-item">
                        <span>👍</span>
                        <span class="article-likes-count" id="card-likes-${article.id}">${article.likes}</span>
                    </span>
                    <span class="stat-item">
                        <span>💬</span>
                        <span class="article-comments-count" id="card-comments-${article.id}">${article.commentsCount}</span>
                    </span>
                </div>
            </header>
            <div class="article-card-content">
                <p class="article-card-excerpt">${Utils.truncateText(article.content)}</p>
            </div>
            <div class="article-card-actions">
                <button class="btn btn--secondary btn--sm card-like-btn" onclick="handleCardLike(${article.id}, this)" data-article-id="${article.id}">
                    👍 Нравится
                </button>
                <button class="btn btn--primary btn--sm read-more-btn" onclick="navigateToArticle(${article.id})">
                    Читать полностью
                </button>
            </div>
        </div>
    `;
}

function renderArticleCards(articles) {
    const grid = document.getElementById('articles-grid');
    if (!grid) return;

    if (articles.length === 0) {
        grid.innerHTML = '<div class="no-articles">Статьи не найдены</div>';
        return;
    }

    grid.innerHTML = articles.map(article => renderArticleCard(article)).join('');
}

function renderArticle(article) {
    const content = document.getElementById('article-content');
    if (!content) return;

    content.querySelector('.article-title').textContent = article.title;
    content.querySelector('.article-author').textContent = article.author;
    content.querySelector('.article-date').textContent = Utils.formatDate(article.date);
    content.querySelector('.article-content').innerHTML = `<p>${article.content}</p>`;
    
    const likeBtn = content.querySelector('#article-like-btn');
    const likeCount = likeBtn.querySelector('.like-count');
    if (likeCount) likeCount.textContent = article.likes;
    
    // Устанавливаем ID статьи в dataset кнопки лайка
    likeBtn.dataset.articleId = article.id;

    likeBtn.classList.remove('liked');
    likeBtn.disabled = false;
    likeBtn.dataset.busy = 'false';

    const commentsCount = content.querySelector('.comments-count');
    if (commentsCount) commentsCount.textContent = article.comments ? article.comments.length : 0;

    renderComments(article.comments || []);
    
    const commentForm = document.getElementById('comment-form');
    if (commentForm) {
        commentForm.reset();
    }
    
    Utils.showElement('article-content');
}

function renderComment(comment, articleId) {
    return `
        <div class="comment" data-comment-id="${comment.id}">
            <div class="comment-header">
                <div class="comment-author-info">
                    <span class="comment-author">${comment.author}</span>
                    <span class="comment-date">${Utils.formatDate(comment.date)}</span>
                </div>
                <div class="comment-actions">
                    <button class="comment-like-btn" onclick="handleCommentLike(${articleId}, ${comment.id}, this)">
                        <span>👍</span>
                        <span class="comment-like-count">${comment.likes}</span>
                    </button>
                </div>
            </div>
            <div class="comment-content">${comment.content}</div>
        </div>
    `;
}

function renderComments(comments) {
    const commentsList = document.getElementById('comments-list');
    if (!commentsList) return;

    if (comments.length === 0) {
        commentsList.innerHTML = '<div class="no-comments">Комментариев пока нет. Будьте первым!</div>';
        return;
    }

    commentsList.innerHTML = comments.map(comment => 
        renderComment(comment, AppState.currentArticleId)
    ).join('');
}

// Основные функции
async function loadArticles() {
    Utils.setLoading('home', true);
    Utils.hideElement('home-error');

    try {
        const articles = await API.getArticles();
        AppState.articles = articles;
        AppState.articlesLoaded = true;
        renderArticleCards(articles);
        Utils.setLoading('home', false);
    } catch (error) {
        Utils.setLoading('home', false);
        Utils.showElement('home-error');
        console.error('Ошибка загрузки статей:', error);
    }
}

async function loadArticle(articleId) {
    Utils.setLoading('article', true);
    Utils.hideElement('article-error');
    Utils.hideElement('article-content');

    try {
        const article = await API.getArticle(articleId);
        AppState.currentArticle = article;
        renderArticle(article);
        Utils.setLoading('article', false);
    } catch (error) {
        Utils.setLoading('article', false);
        Utils.showElement('article-error');
        console.error('Ошибка загрузки статьи:', error);
    }
}

// Инициализация приложения
document.addEventListener('DOMContentLoaded', function() {
    console.log('🌱 Садовод-Профи загружается...');
    
    // Инициализация обработчиков статических элементов
    const headerBtn = document.getElementById('header-home-btn');
    if (headerBtn) {
        headerBtn.onclick = navigateToHome;
    }

    const backBtn = document.getElementById('back-to-home-btn');
    if (backBtn) {
        backBtn.onclick = navigateToHome;
    }

    const commentForm = document.getElementById('comment-form');
    if (commentForm) {
        commentForm.onsubmit = handleAddComment;
    }

    const retryArticlesBtn = document.getElementById('retry-articles-btn');
    if (retryArticlesBtn) {
        retryArticlesBtn.onclick = loadArticles;
    }

    const retryArticleBtn = document.getElementById('retry-article-btn');
    if (retryArticleBtn) {
        retryArticleBtn.onclick = () => {
            if (AppState.currentArticleId) {
                loadArticle(AppState.currentArticleId);
            }
        };
    }
    
    // Запуск приложения
    navigateToHome();
    
    console.log('✅ Садовод-Профи готов к работе!');
});