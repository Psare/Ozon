# Ozon/UrlShorter
docker не работает ;- )
grpc не реализован
отдельные go test на данный момент не реализован, на postgresql работает лучше чем на im-memory
чтобы проверить локально можно изменить мейн и добавить 
storage := flag.String("storage", "in-memory", "Storage type: in-memory or postgres")
	flag.Parse()
и изменить датубазу
