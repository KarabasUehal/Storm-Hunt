import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

export default function CallbackPage() {
  const navigate = useNavigate();

  useEffect(() => {
    // Просто редирект после колбэка
    navigate('/');
  }, [navigate]);

  return <div>Вход... Пожалуйста, подождите.</div>;
}