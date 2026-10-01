package rt;

/** core.map.make: a new empty map. */
public final class MapMake {
    private MapMake() {}

    public static GoMap makeMap(GoMap.KeyOf keyOf) {
        return new GoMap(keyOf);
    }

}
